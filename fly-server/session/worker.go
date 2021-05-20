package session

import "github.com/ngagnon/fly-server/wire"

func runCommands(cb CommandHandler, s *S) {
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
		case arr := <-s.commands:
			if arr.Values[0] == outMarker {
				s.out <- arr.Values[1]
			} else if string(arr.Values[0].(*wire.Blob).Data) == "QUIT" {
				s.out <- wire.OK
				close(s.out)
				return
			} else {
				response := cb(arr, s)
				s.out <- response
			}
		}
	}
}
