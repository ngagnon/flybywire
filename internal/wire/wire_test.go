package wire

import (
	"bytes"
	"errors"
	"testing"
)

func TestIOError(t *testing.T) {
	buf := new(bytes.Buffer)
	reader := NewReader(buf)
	_, err := reader.Read()

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
		cmd = append(cmd, NewString(v))
	}

	if err := NewArray(cmd).WriteTo(buf); err != nil {
		t.Fatal(err)
	}

	reader := NewReader(buf)
	value, err := reader.Read()

	if err != nil {
		t.Fatal(err)
	}

	arr, isArray := value.(*Array)

	if !isArray {
		t.Fatalf("Expected value to be an array, was %s", value.Name())
	}

	if len(arr.Values) != len(expected) {
		t.Fatalf("Expected payload to have length %d, was %d", len(expected), len(arr.Values))
	}

	for i, v := range arr.Values {
		str, ok := v.(*String)

		if !ok {
			t.Fatalf("Expected item %d to be a blob, was %s", i, v.Name())
		}

		if str.Value != expected[i] {
			t.Fatalf("Expected item %d to be '%s', was '%s'", i, expected[i], str.Value)
		}
	}
}

func TestTaggedBlob(t *testing.T) {
	buf := new(bytes.Buffer)
	payload := []byte("Hello, world!")
	blob := NewBlob(payload)

	if err := NewTaggedValue(blob, "1").WriteTo(buf); err != nil {
		t.Fatal(err)
	}

	reader := NewReader(buf)
	value, err := reader.Read()

	if err != nil {
		t.Fatal(err)
	}

	tagged, ok := value.(*TaggedValue)

	if !ok {
		t.Fatalf("Expected value to be tag, was %s", value.Name())
	}

	if tagged.Tag != "1" {
		t.Fatalf("Expected tag to be 1, was %s", tagged.Tag)
	}

	blob, ok = tagged.Value.(*Blob)

	if !ok {
		t.Fatalf("Expected payload to be a blob, was %s", tagged.Value.Name())
	}

	if string(blob.Data) != string(payload) {
		t.Fatalf("Expected payload to be '%s', was '%s'", payload, blob.Data)
	}
}

func TestLargeBlob(t *testing.T) {
	maxSize := 32 * 1024

	buf := new(bytes.Buffer)
	payload := make([]byte, maxSize)

	for i := 0; i < maxSize; i++ {
		payload[i] = 'x'
	}

	blob := NewBlob(payload)

	if err := NewTaggedValue(blob, "1").WriteTo(buf); err != nil {
		t.Fatal(err)
	}

	reader := NewReader(buf)
	reader.MaxBlobSize = maxSize
	value, err := reader.Read()

	if err != nil {
		t.Fatal(err)
	}

	tagged, ok := value.(*TaggedValue)

	if !ok {
		t.Fatalf("Expected value to be tag, was %s", value.Name())
	}

	blob, ok = tagged.Value.(*Blob)

	if !ok {
		t.Fatalf("Expected payload to be a blob, was %s", tagged.Value.Name())
	}

	buf = new(bytes.Buffer)
	payload = append(payload, 'x')
	blob = NewBlob(payload)

	if err := NewTaggedValue(blob, "1").WriteTo(buf); err != nil {
		t.Fatal(err)
	}

	reader = NewReader(buf)
	reader.MaxBlobSize = maxSize
	_, err = reader.Read()
	t.Logf("Payload size: %d", len(payload))

	if !errors.Is(err, ErrFormat) {
		t.Fatalf("Expected format error, got %v", err)
	}
}

func TestTaggedNull(t *testing.T) {
	buf := new(bytes.Buffer)

	if err := NewTaggedValue(Null, "5").WriteTo(buf); err != nil {
		t.Fatal(err)
	}

	reader := NewReader(buf)
	value, err := reader.Read()

	if err != nil {
		t.Fatal(err)
	}

	tagged, ok := value.(*TaggedValue)

	if !ok {
		t.Fatalf("Expected value to be tag, was %s", value.Name())
	}

	if tagged.Tag != "5" {
		t.Fatalf("Expected tag to be 5, was %s", tagged.Tag)
	}

	if tagged.Value != Null {
		t.Fatalf("Expected payload to be null, was %s", tagged.Value.Name())
	}
}
