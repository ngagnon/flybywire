package wire

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

type Value interface {
	WriteTo(io.Writer) error
	Name() string
}

type StreamHeader struct {
	id int
}

type null struct{}

type Array struct {
	Values []Value
}

type Bool struct {
	Value bool
}

type String struct {
	val string
}

type Blob struct {
	Data []byte
}

type Integer struct {
	Value int
}

type Error struct {
	code string
	msg  string
}

type Map struct {
	m map[string]Value
}

type Frame struct {
	StreamId *int
	Payload  Value
}

var OK = &String{val: "OK"}
var Null = &null{}
var ErrFormat = errors.New("Protocol error")
var ErrIO = errors.New("I/O error")

func ReadFrame(r *bufio.Reader) (Frame, error) {
	val, err := readValue(r)

	if err != nil {
		return Frame{}, err
	}

	if header, ok := val.(*StreamHeader); ok {
		return readStreamFrame(header.id, r)
	}

	if cmd, ok := val.(*Array); ok {
		return readCommandFrame(cmd)
	}

	return Frame{}, fmt.Errorf("%w: unexpected %s", ErrFormat, val.Name())
}

func readStreamFrame(id int, r *bufio.Reader) (Frame, error) {
	payload, err := readValue(r)

	if err != nil {
		return Frame{}, err
	}

	_, isBlob := payload.(*Blob)

	if !isBlob && payload != Null {
		err = fmt.Errorf("%w: invalid stream frame, unexpected %s", ErrFormat, payload.Name())
		return Frame{}, err
	}

	return Frame{StreamId: &id, Payload: payload}, nil
}

func readCommandFrame(cmd *Array) (Frame, error) {
	if err := validateCommand(cmd); err != nil {
		return Frame{}, err
	}

	return Frame{Payload: cmd}, nil
}

func validateCommand(arr *Array) error {
	for _, item := range arr.Values {
		if _, ok := item.(*Blob); !ok {
			return fmt.Errorf("%w: invalid command, unexpected %s", ErrFormat, item.Name())
		}
	}

	return nil
}

func readValue(r *bufio.Reader) (Value, error) {
	b, err := r.ReadByte()

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIO, err)
	}

	if b == '_' {
		b, err := r.ReadByte()

		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrIO, err)
		}

		if b != '\n' {
			return nil, fmt.Errorf("%w: unexpected symbol %c, was expecting new line", ErrFormat, rune(b))
		}

		return Null, nil
	}

	if b == '>' || b == '*' || b == '$' {
		// @TODO: readSize might return an IO error, which we wouldn't want to bubble up as a protocol error
		size, err := readSize(r)

		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFormat, err)
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

	return nil, fmt.Errorf("%w: unexpected symbol %c", ErrFormat, rune(b))
}

func handleStreamHeader(id int) (*StreamHeader, error) {
	return &StreamHeader{id: id}, nil
}

func handleArray(r *bufio.Reader, len int) (*Array, error) {
	arr := &Array{
		Values: make([]Value, len),
	}

	for i := 0; i < len; i++ {
		val, err := readValue(r)

		if err != nil {
			return nil, err
		}

		arr.Values[i] = val
	}

	return arr, nil
}

func handleBlob(r *bufio.Reader, size int) (*Blob, error) {
	buf := make([]byte, size)
	_, err := io.ReadFull(r, buf)

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIO, err)
	}

	b, err := r.ReadByte()

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIO, err)
	}

	if b != '\n' {
		return nil, fmt.Errorf("%w: unexpected symbol %c, was expected new line", ErrFormat, rune(b))
	}

	return &Blob{Data: buf}, nil
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
		return nil, fmt.Errorf("%w: %v", ErrIO, err)
	}

	line = bytes.TrimRight(line, "\n")
	return line, nil
}

func NewError(code string, format string, v ...interface{}) *Error {
	msg := fmt.Sprintf(format, v...)
	return &Error{code: code, msg: msg}
}

func NewString(val string) *String {
	return &String{val: val}
}

func NewBlob(data []byte) *Blob {
	return &Blob{Data: data}
}

func NewInteger(val int) *Integer {
	return &Integer{Value: val}
}

func NewBoolean(val bool) *Bool {
	return &Bool{Value: val}
}

func NewMap(m map[string]Value) *Map {
	return &Map{m: m}
}

func NewArray(a []Value) *Array {
	return &Array{Values: a}
}

func NewStreamHeader(id int) *StreamHeader {
	return &StreamHeader{id: id}
}

func (b *Bool) WriteTo(w io.Writer) error {
	out := "#f\n"

	if b.Value {
		out = "#t\n"
	}

	_, err := fmt.Fprint(w, out)
	return err
}

func (s *String) WriteTo(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, "+%s\n", s.val)
	return
}

func (s *Blob) WriteTo(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, "$%d\n%s\n", len(s.Data), s.Data)
	return
}

func (i *Integer) WriteTo(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, ":%d\n", i.Value)
	return
}

func (i *StreamHeader) WriteTo(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, ">%d\n", i.id)
	return
}

func (e *Error) WriteTo(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, "-%s %s\n", e.code, e.msg)
	return
}

func (n *null) WriteTo(w io.Writer) (err error) {
	_, err = fmt.Fprint(w, "_\n")
	return
}

func (a *Array) WriteTo(w io.Writer) error {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "*%d\n", len(a.Values))

	for _, v := range a.Values {
		v.WriteTo(buf)
	}

	_, err := buf.WriteTo(w)
	return err
}

func (m *Map) WriteTo(w io.Writer) error {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%%%d\n", len(m.m))

	for k, v := range m.m {
		ks := String{val: k}
		ks.WriteTo(buf)
		v.WriteTo(buf)
	}

	_, err := buf.WriteTo(w)
	return err
}

func (h *StreamHeader) Name() string {
	return "stream header"
}

func (n *null) Name() string {
	return "null"
}

func (a *Array) Name() string {
	return "array"
}

func (b *Bool) Name() string {
	return "boolean"
}

func (s *String) Name() string {
	return "string"
}

func (b *Blob) Name() string {
	return "blob"
}

func (i *Integer) Name() string {
	return "integer"
}

func (e *Error) Name() string {
	return "error"
}

func (m *Map) Name() string {
	return "map"
}
