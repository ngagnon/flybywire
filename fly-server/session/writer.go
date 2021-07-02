package session

import (
	"errors"
	"io"

	log "github.com/ngagnon/fly-server/logging"
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
			if err != drain {
				log.Debugf("Connection terminated due to write error: %v", err)
			}

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
