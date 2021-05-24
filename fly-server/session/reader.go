package session

import (
	"bufio"
	"errors"
	"io"

	"github.com/ngagnon/fly-server/wire"
)

var outMarker = wire.NewString("OUT")

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

		frame, err := wire.ReadFrame(reader)

		if errors.Is(err, wire.ErrFormat) {
			protoErr := wire.NewError("PROTO", err.Error())
			s.commands <- wire.NewArray([]wire.Value{outMarker, protoErr})
			continue
		}

		if err != nil {
			s.terminate <- struct{}{}
			break
		}

		if frame.StreamId == nil {
			arr := frame.Payload.(*wire.Array)
			s.commands <- arr
		} else {
			stream, ok := s.getStream(*frame.StreamId)

			if !ok {
				argErr := wire.NewError("ARG", "Stream is closed")
				errFrame := wire.NewStreamFrame(*frame.StreamId, argErr)
				s.commands <- wire.NewArray([]wire.Value{outMarker, errFrame})
				continue
			}

			if stream.mode() != write {
				argErr := wire.NewError("ARG", "Stream is not open for writing")
				errFrame := wire.NewStreamFrame(*frame.StreamId, argErr)
				s.commands <- wire.NewArray([]wire.Value{outMarker, errFrame})
				continue
			}

			writeStream := stream.(*writeStream)

			if blob, isBlob := frame.Payload.(*wire.Blob); isBlob {
				writeStream.frames <- newDataFrame(blob.Data)
			} else if frame.Payload == wire.Null {
				writeStream.frames <- newFinishFrame()
			} else {
				protoErr := wire.NewError("PROTO", "Expected blob or null after stream header, got %s", frame.Payload.Name())
				s.commands <- wire.NewArray([]wire.Value{outMarker, protoErr})
			}
		}
	}

	// @TODO: log err?
}
