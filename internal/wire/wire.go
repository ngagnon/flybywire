package wire

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

type Value interface {
	WriteTo(io.Writer) error
	Name() string
}

type TaggedValue struct {
	Tag   string
	Value Value
}

type null struct{}

type Array struct {
	Values []Value
}

type Table struct {
	RowCount int
	ColCount int
	Data     []Value
}

type Bool struct {
	Value bool
}

type String struct {
	Value string
}

type Blob struct {
	Data []byte
}

type Integer struct {
	Value int
}

type Error struct {
	Code    string
	Message string
}

type Map struct {
	m map[string]Value
}

var OK = &String{Value: "OK"}
var Null = &null{}
var ErrFormat = errors.New("Protocol error")
var ErrIO = errors.New("I/O error")

type WireReader struct {
	MaxBlobSize int

	r *bufio.Reader
}

func NewReader(r io.Reader) *WireReader {
	bufReader, ok := r.(*bufio.Reader)

	if !ok {
		bufReader = bufio.NewReader(r)
	}

	return &WireReader{
		MaxBlobSize: math.MaxInt64,
		r:           bufReader,
	}
}

func (r *WireReader) Read() (Value, error) {
	return readValue(r, true)
}

func ReadValue(r io.Reader) (Value, error) {
	reader := NewReader(r)
	return reader.Read()
}

func readValue(r *WireReader, canBeTag bool) (Value, error) {
	b, err := r.r.ReadByte()

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIO, err)
	}

	switch b {
	case '_':
		return handleNull(r.r)
	case '#':
		return handleBoolean(r.r)
	case '+':
		return handleString(r.r)
	case '-':
		return handleError(r.r)
	case '@':
		return handleTag(r, canBeTag)
	case '=':
		return handleTable(r)
	case '*':
		fallthrough
	case '$':
		fallthrough
	case ':':
		size, err := readSize(r.r)

		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFormat, err)
		}

		switch b {
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

func handleNull(r *bufio.Reader) (Value, error) {
	b, err := r.ReadByte()

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIO, err)
	}

	if b != '\n' {
		return nil, fmt.Errorf("%w: unexpected symbol %c, was expecting new line", ErrFormat, rune(b))
	}

	return Null, nil
}

func handleTable(r *WireReader) (Value, error) {
	rows, cols, err := readTableSize(r.r)

	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFormat, err)
	}

	t := &Table{
		RowCount: 0,
		ColCount: cols,
	}

	for i := 0; i < rows; i++ {
		row := make([]Value, cols)

		for j, _ := range row {
			val, err := readValue(r, false)

			if err != nil {
				return nil, err
			}

			row[j] = val
		}

		t.Add(row)
	}

	return t, nil
}

func readTableSize(r *bufio.Reader) (rows int, cols int, err error) {
	line, err := nextLine(r)

	if err != nil {
		return
	}

	i := strings.Index(string(line), ",")

	if i == -1 {
		return 0, 0, fmt.Errorf("invalid table size: %s", string(line))
	}

	if rows, err = strconv.Atoi(string(line[0:i])); err != nil {
		return 0, 0, fmt.Errorf("invalid table size: %s", string(line))
	}

	if cols, err = strconv.Atoi(string(line[i+1:])); err != nil {
		return 0, 0, fmt.Errorf("invalid table size: %s", string(line))
	}

	return
}

func handleBoolean(r *bufio.Reader) (Value, error) {
	sym, err := r.ReadByte()

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIO, err)
	}

	if sym != 't' && sym != 'f' {
		return nil, fmt.Errorf("%w: unexpected symbol %c, was expecting t or f after #", ErrFormat, rune(sym))
	}

	b, err := r.ReadByte()

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIO, err)
	}

	if b != '\n' {
		return nil, fmt.Errorf("%w: unexpected symbol %c, was expecting new line", ErrFormat, rune(b))
	}

	return NewBoolean(sym == 't'), nil
}

func handleString(r *bufio.Reader) (Value, error) {
	buf, err := r.ReadBytes('\n')

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIO, err)
	}

	buf = buf[:len(buf)-1]

	return NewString(string(buf)), nil
}

func handleError(r *bufio.Reader) (Value, error) {
	buf, err := r.ReadBytes('\n')

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIO, err)
	}

	buf = buf[:len(buf)-1]
	i := strings.Index(string(buf), " ")

	if i == -1 {
		return nil, fmt.Errorf("%w: error should have at least one space", ErrFormat)
	}

	return NewError(string(buf[0:i]), string(buf[i+1:])), nil
}

func handleTag(r *WireReader, canBeTag bool) (Value, error) {
	if !canBeTag {
		return nil, fmt.Errorf("%w: unexpected tag", ErrFormat)
	}

	buf, err := r.r.ReadBytes('\n')

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIO, err)
	}

	buf = buf[:len(buf)-1]
	val, err := readValue(r, false)

	if err != nil {
		return nil, err
	}

	return NewTaggedValue(val, string(buf)), nil
}

func handleArray(r *WireReader, len int) (*Array, error) {
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

func handleBlob(r *WireReader, size int) (*Blob, error) {
	if size > r.MaxBlobSize {
		return nil, fmt.Errorf("%w: blobs cannot exceed %d in length", ErrFormat, r.MaxBlobSize)
	}

	buf := make([]byte, size)
	_, err := io.ReadFull(r.r, buf)

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIO, err)
	}

	b, err := r.r.ReadByte()

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
	return &Error{Code: code, Message: msg}
}

func NewString(val string) *String {
	return &String{Value: val}
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

func NewTaggedValue(val Value, tag string) *TaggedValue {
	return &TaggedValue{Tag: tag, Value: val}
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
	_, err = fmt.Fprintf(w, "+%s\n", s.Value)
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

func (f *TaggedValue) WriteTo(w io.Writer) (err error) {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "@%s\n", f.Tag)
	f.Value.WriteTo(buf)
	_, err = buf.WriteTo(w)
	return
}

func (e *Error) WriteTo(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, "-%s %s\n", e.Code, e.Message)
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

func (t *Table) WriteTo(w io.Writer) error {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "=%d,%d\n", t.RowCount, t.ColCount)

	for _, v := range t.Data {
		v.WriteTo(buf)
	}

	_, err := buf.WriteTo(w)
	return err
}

func (m *Map) WriteTo(w io.Writer) error {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%%%d\n", len(m.m))

	for k, v := range m.m {
		ks := String{Value: k}
		ks.WriteTo(buf)
		v.WriteTo(buf)
	}

	_, err := buf.WriteTo(w)
	return err
}

func (h *TaggedValue) Name() string {
	return "tagged value"
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

func (t *Table) Name() string {
	return "table"
}

func (t *Table) Add(row []Value) {
	if len(t.Data) == 0 {
		t.ColCount = len(row)
		t.RowCount = 0
	}

	t.Data = append(t.Data, row...)
	t.RowCount++
}

func (t *Table) Row(id int) []Value {
	if id < 0 || id >= t.RowCount {
		return nil
	}

	return t.Data[id*t.ColCount : (id+1)*t.ColCount]
}
