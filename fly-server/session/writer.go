package session

import "io"

func handleWrites(conn io.Writer, s *S) {
	s.waitGroup.Add(1)
	defer s.waitGroup.Done()

	for {
		select {
		case <-s.done:
			return
		default:
		}

		select {
		case <-s.done:
			return
		case val := <-s.out:
			if val == nil {
				// out channel was closed. now ready to terminate the session
				s.terminate <- struct{}{}
				return
			}

			err := val.WriteTo(conn)

			if err != nil {
				s.terminate <- struct{}{}
				break
			}
		}
	}

	// @TODO: log err?
}
