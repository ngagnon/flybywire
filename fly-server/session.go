package main

import (
	"bufio"
	"bytes"
	"errors"
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

type respType int

const (
	RespNull respType = iota
	RespBulkString
	RespSimpleString
	RespErrorString
	RespBoolean
	RespMap
)

type respValue struct {
	valueType respType
	value     interface{}
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

func (s *session) writeSimpleString(str string) (err error) {
	_, err = fmt.Fprintf(s, "+%s\n", str)
	return
}

func (s *session) writeError(code string, msg string) (err error) {
	_, err = fmt.Fprintf(s, "-%s %s\n", code, msg)
	return
}

func (s *session) writeOK() error {
	return s.writeSimpleString("OK")
}

func (s *session) writeNull() (err error) {
	_, err = fmt.Fprint(s, "_\n")
	return
}

func (s *session) writeBool(b bool) (err error) {
	out := "0\n"

	if b {
		out = "1\n"
	}

	_, err = fmt.Fprint(s, out)
	return
}

func (s *session) writeMap(m map[string]respValue) error {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%%%d\n", len(m))

	prevWriter := s.writer
	s.writer = buf

	for k, v := range m {
		s.writeSimpleString(k)

		switch v.valueType {
		case RespSimpleString:
			s.writeSimpleString(v.value.(string))
		case RespBoolean:
			s.writeBool(v.value.(bool))
		default:
			msg := fmt.Sprintf("writeMap: Unsupported RESP type: %d", v.valueType)
			panic(errors.New(msg))
		}
	}

	s.writer = prevWriter
	_, err := buf.WriteTo(prevWriter)
	return err
}

func (err *protocolError) Error() string {
	return err.msg
}
