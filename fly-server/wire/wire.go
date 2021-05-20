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

type StreamFrame struct {
	id    int
	Value Value
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
	val, err := readValue(r, true)

	if err != nil {
		return Frame{}, err
	}

	if stream, ok := val.(*StreamFrame); ok {
		return readStreamFrame(stream, r)
	}

	if cmd, ok := val.(*Array); ok {
		return readCommandFrame(cmd)
	}

	return Frame{}, fmt.Errorf("%w: unexpected %s", ErrFormat, val.Name())
}

func readStreamFrame(frame *StreamFrame, r *bufio.Reader) (Frame, error) {
	payload := frame.Value
	_, isBlob := payload.(*Blob)

	if !isBlob && payload != Null {
		err := fmt.Errorf("%w: invalid stream frame, unexpected %s", ErrFormat, payload.Name())
		return Frame{}, err
	}

	return Frame{StreamId: &frame.id, Payload: payload}, nil
}

func readCommandFrame(cmd *Array) (Frame, error) {
	if err := validateCommand(cmd); err != nil {
		return Frame{}, err
	}

	return Frame{Payload: cmd}, nil
}

func validateCommand(arr *Array) error {
	if len(arr.Values) == 0 {
		return fmt.Errorf("%w: unexpected empty array", ErrFormat)
	}

	if _, ok := arr.Values[0].(*Blob); !ok {
		return fmt.Errorf("%w: command name not a string, got %s instead", ErrFormat, arr.Values[0].Name())
	}

	return nil
}

func readValue(r *bufio.Reader, canBeStream bool) (Value, error) {
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

	if b == '>' || b == '*' || b == '$' || b == ':' {
		size, err := readSize(r)

		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFormat, err)
		}

		switch b {
		case '>':
			if !canBeStream {
				return nil, fmt.Errorf("%w: unexpected stream header", ErrFormat)
			}

			return handleStreamFrame(size, r)
		case '*':
			return handleArray(r, size)
		case '$':
			return handleBlob(r, size)
		case ':':
			return &Integer{Value: size}, nil
		}
	}

	return nil, fmt.Errorf("%w: unexpected symbol %c", ErrFormat, rune(b))
}

func handleStreamFrame(id int, r *bufio.Reader) (*StreamFrame, error) {
	val, err := readValue(r, false)

	if err != nil {
		return nil, err
	}

	return NewStreamFrame(id, val), nil
}

func handleArray(r *bufio.Reader, len int) (*Array, error) {
	arr := &Array{
		Values: make([]Value, len),
	}

	for i := 0; i < len; i++ {
		val, err := readValue(r, false)

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

func NewStreamFrame(id int, val Value) *StreamFrame {
	return &StreamFrame{id: id, Value: val}
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

func (f *StreamFrame) WriteTo(w io.Writer) (err error) {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, ">%d\n", f.id)
	f.Value.WriteTo(buf)
	_, err = buf.WriteTo(w)
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

func (h *StreamFrame) Name() string {
	return "stream frame"
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
