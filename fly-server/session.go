package main

import (
	"bufio"
	"bytes"
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
	writeErr   chan error
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

type command struct {
	name string
	args []string
}

type protocolError struct {
	msg string
}

type respValue interface {
	writeTo(io.Writer) error
}

type respNull struct{}

type respBool struct {
	val bool
}

type respString struct {
	val string
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

func newSession(conn net.Conn) *session {
	return &session{
		terminated: false,
		user:       "",
		reader:     bufio.NewReader(conn),
		out:        make(chan respValue, 10),
		writeErr:   make(chan error),
	}
}

func (s *session) nextCommand() (command, error) {
	for {
		numElems, err := readArrayHeader(s.reader)

		if err != nil {
			return command{}, err
		}

		arr := make([]string, numElems)

		for i := 0; i < numElems; i++ {
			size, err := readBlobHeader(s.reader)

			if err != nil {
				return command{}, err
			}

			buf := make([]byte, size)
			io.ReadFull(s.reader, buf)

			if b, _ := s.reader.ReadByte(); b != '\n' {
				msg := fmt.Sprintf("Protocol error: unexpected symbol '%c', was expecting new line", rune(b))
				return command{}, &protocolError{msg: msg}
			}

			arr[i] = string(buf)
		}

		return command{
			name: arr[0],
			args: arr[1:],
		}, nil
	}
}

func readArrayHeader(r *bufio.Reader) (count int, err error) {
	return readSizeHeader(r, "*")
}

func readBlobHeader(r *bufio.Reader) (count int, err error) {
	return readSizeHeader(r, "$")
}

func readSizeHeader(r *bufio.Reader, prefix string) (count int, err error) {
	line, err := nextLine(r)

	if err != nil {
		return 0, err
	}

	if !bytes.HasPrefix(line, []byte(prefix)) {
		msg := fmt.Sprintf("Protocol error: unexpected symbol '%c'", rune(line[0]))
		return 0, &protocolError{msg: msg}
	}

	line = bytes.TrimPrefix(line, []byte(prefix))

	var n int

	if n, err = strconv.Atoi(string(line)); err != nil {
		msg := fmt.Sprintf("Protocol error: %s", err)
		return 0, &protocolError{msg: msg}
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

func (s *session) writeString(str string) (err error) {
	return s.write(&respString{val: str})
}

func (s *session) writeError(code string, msg string) (err error) {
	return s.write(&respError{code: code, msg: msg})
}

func (s *session) writeInt(i int) (err error) {
	return s.write(&respInteger{val: i})
}

func (s *session) writeOK() error {
	return s.writeString("OK")
}

func (s *session) writeNull() (err error) {
	return s.write(&respNull{})
}

func (s *session) writeMap(m map[string]respValue) error {
	return s.write(&respMap{m: m})
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

func (i *respInteger) writeTo(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, ":%d\n", i.val)
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

func (err *protocolError) Error() string {
	return err.msg
}
