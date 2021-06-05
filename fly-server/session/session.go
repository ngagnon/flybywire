package session

import (
	"net"
	"sync"

	"github.com/ngagnon/fly-server/wire"
)

type S struct {
	terminate  chan struct{}
	waitGroup  *sync.WaitGroup
	done       chan struct{}
	out        chan wire.Value
	commands   chan *wire.Array
	streams    [16]stream
	streamLock sync.RWMutex
	user       string
}

type CommandHandler func(cmd *wire.Array, session *S) (response wire.Value)

func Handle(conn net.Conn, cb CommandHandler) {
	session := &S{
		terminate: make(chan struct{}, 3),
		done:      make(chan struct{}),
		waitGroup: &sync.WaitGroup{},
		out:       make(chan wire.Value, 5),
		commands:  make(chan *wire.Array, 5),
	}

	go handleReads(conn, session)
	go handleWrites(conn, session)
	go runCommands(cb, session)

	<-session.terminate
	close(session.done)
	session.waitGroup.Wait()
	conn.Close()
}

func (s *S) CurrentUser() string {
	return s.user
}

func (s *S) SetUser(user string) {
	s.user = user
}
