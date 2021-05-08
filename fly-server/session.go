package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
)

type session struct {
	terminated bool
	user       string
	reader     *bufio.Reader
	out        chan respValue
	streams    [16]*stream
	streamLock sync.RWMutex
}

type stream struct {
	finish    chan struct{}
	cancel    chan struct{}
	data      chan []byte
	finalPath string
	file      *os.File
}

type frame struct {
	streamId *int
	val      respValue
}

type respValue interface {
	writeTo(io.Writer) error
	name() string
}

type respStreamHeader struct {
	id int
}

type respNull struct{}

type respArray struct {
	values []respValue
}

type respBool struct {
	val bool
}

type respString struct {
	val string
}

type respBlob struct {
	val []byte
}

type respInteger struct {
	val int
}

type respError struct {
	code string
	msg  string
}

type respMap struct {
	m map[string]respValue
}

var ErrProtocol = errors.New("Protocol error")
var RespOK = &respString{val: "OK"}

func newSession(conn net.Conn) *session {
	return &session{
		terminated: false,
		user:       "",
		reader:     bufio.NewReader(conn),
		out:        make(chan respValue, 10),
	}
}

// Returns a valid frame (command or stream)
// Returns an error if an IO error occurred on read
func (s *session) nextFrame() (frame, error) {
	for {
		val, err := readValue(s.reader)

		if errors.Is(err, ErrProtocol) {
			s.out <- newError("PROTO", err.Error())
			continue
		}

		if err != nil {
			return frame{}, err
		}

		if header, ok := val.(*respStreamHeader); ok {
			payload, err := readValue(s.reader)

			if errors.Is(err, ErrProtocol) {
				s.out <- newError("PROTO", err.Error())
				continue
			}

			if err != nil {
				return frame{}, err
			}

			_, isBlob := payload.(*respBlob)
			_, isNull := payload.(*respNull)

			if !isBlob && !isNull {
				msg := fmt.Sprintf("Protocol error: invalid stream frame, unexpected %s", payload.name())
				s.out <- newError("PROTO", msg)
				continue
			}

			return frame{streamId: &header.id, val: payload}, nil
		}

		if cmd, ok := val.(*respArray); ok {
			if t := validCommand(cmd); t != "" {
				msg := fmt.Sprintf("Protocol error: invalid command, unexpected %s", t)
				s.out <- newError("PROTO", msg)
				continue
			}

			return frame{val: cmd}, nil
		}

		msg := fmt.Sprintf("Protocol error: unexpected %s", val.name())
		s.out <- newError("PROTO", msg)
	}
}

func validCommand(arr *respArray) string {
	for _, item := range arr.values {
		if _, ok := item.(*respBlob); !ok {
			return item.name()
		}
	}

	return ""
}

func readValue(r *bufio.Reader) (respValue, error) {
	b, err := r.ReadByte()

	if err != nil {
		return nil, err
	}

	if b == '_' {
		b, err := r.ReadByte()

		if err != nil {
			return nil, err
		}

		if b != '\n' {
			return nil, fmt.Errorf("%w: unexpected symbol %c, was expecting new line", ErrProtocol, rune(b))
		}

		return &respNull{}, nil
	}

	if b == '>' || b == '*' || b == '$' {
		// @TODO: readSize might return an IO error, which we wouldn't want to bubble up as a protocol error
		size, err := readSize(r)

		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrProtocol, err)
		}

		switch b {
		case '>':
			return handleStreamHeader(size)
		case '*':
			return handleArray(r, size)
		case '$':
			return handleBlob(r, size)
		}
	}

	return nil, fmt.Errorf("%w: unexpected symbol %c", ErrProtocol, rune(b))
}

func handleStreamHeader(id int) (*respStreamHeader, error) {
	return &respStreamHeader{id: id}, nil
}

func handleArray(r *bufio.Reader, len int) (*respArray, error) {
	arr := &respArray{
		values: make([]respValue, len),
	}

	for i := 0; i < len; i++ {
		val, err := readValue(r)

		if err != nil {
			return nil, err
		}

		arr.values[i] = val
	}

	return arr, nil
}

func handleBlob(r *bufio.Reader, size int) (*respBlob, error) {
	buf := make([]byte, size)
	_, err := io.ReadFull(r, buf)

	if err != nil {
		return nil, err
	}

	if b, _ := r.ReadByte(); b != '\n' {
		return nil, fmt.Errorf("%w: unexpected symbol %c, was expected new line", ErrProtocol, rune(b))
	}

	return &respBlob{val: buf}, nil
}

func readSize(r *bufio.Reader) (count int, err error) {
	line, err := nextLine(r)

	if err != nil {
		return 0, err
	}

	var n int

	if n, err = strconv.Atoi(string(line)); err != nil {
		return 0, fmt.Errorf("invalid size: %s", string(line))
	}

	return n, nil
}

func nextLine(r *bufio.Reader) ([]byte, error) {
	for {
		line, err := readLine(r)

		if err != nil {
			return nil, err
		}

		if len(line) > 0 {
			return line, nil
		}
	}
}

func readLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes('\n')

	if err != nil {
		return nil, err
	}

	line = bytes.TrimRight(line, "\n")
	return line, nil
}

func (s *session) getStream(id int) (stream *stream, ok bool) {
	s.streamLock.RLock()
	defer s.streamLock.RUnlock()

	if id < 0 || id >= len(s.streams) {
		return nil, false
	}

	stream = s.streams[id]
	ok = stream != nil
	return
}

func (s *session) addStream(stream *stream) (id int, ok bool) {
	s.streamLock.Lock()
	defer s.streamLock.Unlock()

	id, ok = nextStreamId(s.streams[:])

	if ok {
		s.streams[id] = stream
	}

	return
}

func (s *session) closeStream(id int) {
	s.streamLock.Lock()
	s.streams[id] = nil
	s.streamLock.Unlock()
}

func nextStreamId(streams []*stream) (id int, ok bool) {
	for i := 0; i < len(streams); i++ {
		if streams[i] == nil {
			return i, true
		}
	}

	return 0, false
}

func (s *session) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *session) write(val respValue) error {
	s.out <- val
	return nil
}

func (s *session) writeString(str string) {
	s.write(&respString{val: str})
}

func newError(code string, format string, v ...interface{}) *respError {
	msg := fmt.Sprintf(format, v...)
	return &respError{code: code, msg: msg}
}

func (b *respBool) writeTo(w io.Writer) error {
	out := "#f\n"

	if b.val {
		out = "#t\n"
	}

	_, err := fmt.Fprint(w, out)
	return err
}

func (s *respString) writeTo(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, "+%s\n", s.val)
	return
}

func (s *respBlob) writeTo(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, "$%d\n%s\n", len(s.val), s.val)
	return
}

func (i *respInteger) writeTo(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, ":%d\n", i.val)
	return
}

func (i *respStreamHeader) writeTo(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, ">%d\n", i.id)
	return
}

func (e *respError) writeTo(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, "-%s %s\n", e.code, e.msg)
	return
}

func (n *respNull) writeTo(w io.Writer) (err error) {
	_, err = fmt.Fprint(w, "_\n")
	return
}

func (a *respArray) writeTo(w io.Writer) error {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "*%d\n", len(a.values))

	for _, v := range a.values {
		v.writeTo(buf)
	}

	_, err := buf.WriteTo(w)
	return err
}

func (m *respMap) writeTo(w io.Writer) error {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%%%d\n", len(m.m))

	for k, v := range m.m {
		ks := respString{val: k}
		ks.writeTo(buf)
		v.writeTo(buf)
	}

	_, err := buf.WriteTo(w)
	return err
}

func (h *respStreamHeader) name() string {
	return "stream header"
}

func (n *respNull) name() string {
	return "null"
}

func (a *respArray) name() string {
	return "array"
}

func (b *respBool) name() string {
	return "boolean"
}

func (s *respString) name() string {
	return "string"
}

func (b *respBlob) name() string {
	return "blob"
}

func (i *respInteger) name() string {
	return "integer"
}

func (e *respError) name() string {
	return "error"
}

func (m *respMap) name() string {
	return "map"
}
