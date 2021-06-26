package session

import (
	"errors"
	"io"
)

var done = errors.New("done")
var drain = errors.New("drain")

func handleWrites(conn io.Writer, s *S) {
	s.waitGroup.Add(1)
	defer s.waitGroup.Done()

	for {
		err := handleWrite(conn, s)

		if err == done {
			return
		}

		if err != nil {
			// @TODO: log err? (if != drain)
			s.terminate <- struct{}{}
			return
		}
	}
}

func handleWrite(conn io.Writer, s *S) error {
	select {
	case <-s.done:
		return done
	default:
	}

	select {
	case val := <-s.cmdOut:
		// cmdOut has been closed, which means we should drain cmdOut, then terminate
		if val == nil {
			return drain
		}

		return val.WriteTo(conn)
	default:
	}

	select {
	case <-s.done:
		return done
	case val := <-s.cmdOut:
		// cmdOut has been closed, which means we should drain cmdOut, then terminate
		if val == nil {
			return drain
		}

		return val.WriteTo(conn)
	case val := <-s.dataOut:
		return val.WriteTo(conn)
	}
}
