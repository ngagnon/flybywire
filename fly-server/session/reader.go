package session

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/ngagnon/fly-server/wire"
)

var outMarker = wire.NewString("OUT")

func handleReads(conn net.Conn, s *S) {
	s.waitGroup.Add(1)
	defer s.waitGroup.Done()

	bufReader := bufio.NewReader(conn)
	reader := wire.NewReader(bufReader)
	reader.MaxBlobSize = 32 * 1024

	for {
		select {
		case <-s.done:
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		value, err := reader.Read()

		if errors.Is(err, wire.ErrFormat) {
			protoErr := wire.NewError("PROTO", err.Error())
			s.enqueueResponse(protoErr)
			continue
		}

		if err != nil {
			s.terminate <- struct{}{}
			break
		}

		if tagged, isTagged := value.(*wire.TaggedValue); isTagged {
			handleStreamFrame(tagged, s)
			continue
		}

		if array, isArray := value.(*wire.Array); isArray {
			handleCommand(array, s)
			continue
		}

		s.protocolError("unexpected %s", value.Name())
	}

	// @TODO: log err?
}

func handleCommand(array *wire.Array, s *S) {
	if len(array.Values) == 0 {
		s.protocolError("unexpected empty array")
		return
	}

	if _, ok := array.Values[0].(*wire.String); !ok {
		s.protocolError("command name should be a string, was %s", array.Values[0].Name())
		return
	}

	s.commands <- array
}

func handleStreamFrame(tagged *wire.TaggedValue, s *S) {
	payload := tagged.Value
	blob, isBlob := payload.(*wire.Blob)

	if !isBlob && payload != wire.Null {
		s.protocolError("invalid stream frame, unexpected %s", payload.Name())
		return
	}

	streamId, err := strconv.Atoi(tagged.Tag)

	if err != nil {
		s.protocolError("Protocol error: invalid stream ID: %s", tagged.Tag)
		return
	}

	stream, ok := s.getStream(streamId)

	if !ok {
		s.streamError("ARG", "Stream is closed", tagged.Tag)
		return
	}

	if stream.mode() != write {
		s.streamError("ARG", "Stream is not open for writing", tagged.Tag)
		return
	}

	writeStream := stream.(*writeStream)

	if isBlob {
		writeStream.frames <- newDataFrame(blob.Data)
	} else {
		writeStream.frames <- newFinishFrame()
	}
}

func (s *S) enqueueResponse(val wire.Value) {
	s.commands <- wire.NewArray([]wire.Value{outMarker, val})
}

func (s *S) protocolError(format string, v ...interface{}) {
	msg := fmt.Sprintf("Protocol error: "+format, v...)
	protoErr := wire.NewError("PROTO", msg)
	s.enqueueResponse(protoErr)
}

func (s *S) streamError(code string, msg string, tag string) {
	argErr := wire.NewError(code, msg)
	errFrame := wire.NewTaggedValue(argErr, tag)
	s.enqueueResponse(errFrame)
}
