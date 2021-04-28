package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
)

type session struct {
	terminated bool
	user       string
	reader     *bufio.Reader
	writer     io.Writer
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

func newSession(conn net.Conn) *session {
	return &session{
		terminated: false,
		user:       "",
		reader:     bufio.NewReader(conn),
		writer:     conn,
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

func (s *session) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *session) Write(p []byte) (int, error) {
	return s.writer.Write(p)
}

func (s *session) write(val respValue) error {
	return val.writeTo(s)
}

func (s *session) writeString(str string) (err error) {
	return s.write(&respString{val: str})
}

func (s *session) writeError(code string, msg string) (err error) {
	_, err = fmt.Fprintf(s, "-%s %s\n", code, msg)
	return
}

func (s *session) writeOK() error {
	return s.writeString("OK")
}

func (s *session) writeNull() (err error) {
	return s.write(&respNull{})
}

func (s *session) writeMap(m map[string]respValue) error {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%%%d\n", len(m))

	prevWriter := s.writer
	s.writer = buf

	for k, v := range m {
		s.writeString(k)
		v.writeTo(buf)
	}

	s.writer = prevWriter
	_, err := buf.WriteTo(prevWriter)
	return err
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

func (n *respNull) writeTo(w io.Writer) (err error) {
	_, err = fmt.Fprint(w, "_\n")
	return
}

func (err *protocolError) Error() string {
	return err.msg
}
