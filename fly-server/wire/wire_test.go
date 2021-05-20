package wire

import (
	"bufio"
	"bytes"
	"errors"
	"testing"
)

func TestIOError(t *testing.T) {
	buf := new(bytes.Buffer)
	reader := bufio.NewReader(buf)
	_, err := ReadFrame(reader)

	if !errors.Is(err, ErrIO) {
		t.Fatalf("Expected I/O error, got %v", err)
	}
}

func TestCommandFrame(t *testing.T) {
	buf := new(bytes.Buffer)

	expected := make([]string, 0, 3)
	expected = append(expected, "STREAM")
	expected = append(expected, "R")
	expected = append(expected, "/some/path/file.txt")

	cmd := make([]Value, 0, 3)

	for _, v := range expected {
		cmd = append(cmd, NewBlob([]byte(v)))
	}

	if err := NewArray(cmd).WriteTo(buf); err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(buf)
	frame, err := ReadFrame(reader)

	if err != nil {
		t.Fatal(err)
	}

	if frame.StreamId != nil {
		t.Fatalf("Expected stream ID to be nil, was %d", *frame.StreamId)
	}

	arr, isArray := frame.Payload.(*Array)

	if !isArray {
		t.Fatalf("Expected payload to be an array, was %s", frame.Payload.Name())
	}

	if len(arr.Values) != len(expected) {
		t.Fatalf("Expected payload to have length %d, was %d", len(expected), len(arr.Values))
	}

	for i, v := range arr.Values {
		blob, ok := v.(*Blob)

		if !ok {
			t.Fatalf("Expected item %d to be a blob, was %s", i, v.Name())
		}

		if string(blob.Data) != expected[i] {
			t.Fatalf("Expected item %d to be '%s', was '%s'", i, expected[i], blob.Data)
		}
	}
}

func TestInvalidCommandFrame(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := make([]Value, 0, 3)
	cmd = append(cmd, NewInteger(42))
	cmd = append(cmd, NewBlob([]byte("foo")))
	cmd = append(cmd, NewBlob([]byte("bar")))

	if err := NewArray(cmd).WriteTo(buf); err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(buf)
	_, err := ReadFrame(reader)

	if !errors.Is(err, ErrFormat) {
		t.Fatalf("Expected format error, got %v", err)
	}
}

func TestInvalidStreamFrame(t *testing.T) {
	buf := new(bytes.Buffer)
	payload := NewInteger(42)

	if err := NewStreamFrame(10, payload).WriteTo(buf); err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(buf)
	_, err := ReadFrame(reader)

	if !errors.Is(err, ErrFormat) {
		t.Fatalf("Expected format error, got %v", err)
	}
}

func TestStreamBlobFrame(t *testing.T) {
	buf := new(bytes.Buffer)
	payload := []byte("Hello, world!")
	blob := NewBlob(payload)

	if err := NewStreamFrame(1, blob).WriteTo(buf); err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(buf)
	frame, err := ReadFrame(reader)

	if err != nil {
		t.Fatal(err)
	}

	if frame.StreamId == nil {
		t.Fatal("Expected stream ID to be 1, was nil")
	}

	if *frame.StreamId != 1 {
		t.Fatalf("Expected stream ID to be 1, was %d", *frame.StreamId)
	}

	blob, ok := frame.Payload.(*Blob)

	if !ok {
		t.Fatalf("Expected payload to be a blob, was %s", frame.Payload.Name())
	}

	if string(blob.Data) != string(payload) {
		t.Fatalf("Expected payload to be '%s', was '%s'", payload, blob.Data)
	}
}

func TestStreamNullFrame(t *testing.T) {
	buf := new(bytes.Buffer)

	if err := NewStreamFrame(5, Null).WriteTo(buf); err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(buf)
	frame, err := ReadFrame(reader)

	if err != nil {
		t.Fatal(err)
	}

	if frame.StreamId == nil {
		t.Fatal("Expected stream ID to be 5, was nil")
	}

	if *frame.StreamId != 5 {
		t.Fatalf("Expected stream ID to be 5, was %d", *frame.StreamId)
	}

	if frame.Payload != Null {
		t.Fatalf("Expected payload to be null, was %s", frame.Payload.Name())
	}
}
