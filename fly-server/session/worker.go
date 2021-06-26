package session

import (
	"strings"

	"github.com/ngagnon/fly-server/wire"
)

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
				s.cmdOut <- arr.Values[1]
			} else if commandIsQuit(arr.Values[0]) {
				s.cmdOut <- wire.OK
				close(s.cmdOut) // drain the pending writes
				return
			} else {
				response := cb(arr, s)
				s.cmdOut <- response
			}
		}
	}
}

func commandIsQuit(val wire.Value) bool {
	return strings.ToUpper(val.(*wire.String).Value) == "QUIT"
}
