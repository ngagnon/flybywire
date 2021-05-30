package session

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/ngagnon/fly-server/wire"
)

var outMarker = wire.NewString("OUT")

// @TODO: refactor
func handleReads(conn io.Reader, s *S) {
	s.waitGroup.Add(1)
	defer s.waitGroup.Done()

	reader := bufio.NewReader(conn)

	for {
		select {
		case <-s.done:
			return
		default:
		}

		value, err := wire.ReadValue(reader)

		if errors.Is(err, wire.ErrFormat) {
			protoErr := wire.NewError("PROTO", err.Error())
			s.commands <- wire.NewArray([]wire.Value{outMarker, protoErr})
			continue
		}

		if err != nil {
			s.terminate <- struct{}{}
			break
		}

		if tagged, isTagged := value.(*wire.TaggedValue); isTagged {
			payload := tagged.Value
			blob, isBlob := payload.(*wire.Blob)

			if !isBlob && payload != wire.Null {
				msg := fmt.Sprintf("Protocol error: invalid stream frame, unexpected %s", payload.Name())
				protoErr := wire.NewError("PROTO", msg)
				s.commands <- wire.NewArray([]wire.Value{outMarker, protoErr})
				continue
			}

			streamId, err := strconv.Atoi(tagged.Tag)

			if err != nil {
				msg := fmt.Sprintf("Protocol error: invalid stream ID: %s", tagged.Tag)
				protoErr := wire.NewError("PROTO", msg)
				s.commands <- wire.NewArray([]wire.Value{outMarker, protoErr})
				continue
			}

			stream, ok := s.getStream(streamId)

			if !ok {
				argErr := wire.NewError("ARG", "Stream is closed")
				errFrame := wire.NewTaggedValue(argErr, tagged.Tag)
				s.commands <- wire.NewArray([]wire.Value{outMarker, errFrame})
				continue
			}

			if stream.mode() != write {
				argErr := wire.NewError("ARG", "Stream is not open for writing")
				errFrame := wire.NewTaggedValue(argErr, tagged.Tag)
				s.commands <- wire.NewArray([]wire.Value{outMarker, errFrame})
				continue
			}

			writeStream := stream.(*writeStream)

			if isBlob {
				writeStream.frames <- newDataFrame(blob.Data)
			} else {
				writeStream.frames <- newFinishFrame()
			}

			continue
		}

		if array, isArray := value.(*wire.Array); isArray {
			if len(array.Values) == 0 {
				msg := "Protocol error: unexpected empty array"
				protoErr := wire.NewError("PROTO", msg)
				s.commands <- wire.NewArray([]wire.Value{outMarker, protoErr})
				continue
			}

			if _, ok := array.Values[0].(*wire.String); !ok {
				msg := fmt.Sprintf("Protocol error: command name should be a string, was %s", array.Values[0].Name())
				protoErr := wire.NewError("PROTO", msg)
				s.commands <- wire.NewArray([]wire.Value{outMarker, protoErr})
				continue
			}

			s.commands <- array
			continue
		}

		protoErr := wire.NewError("PROTO", "Protocol error: unexpected %s", value.Name())
		s.commands <- wire.NewArray([]wire.Value{outMarker, protoErr})
	}

	// @TODO: log err?
}
