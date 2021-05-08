package main

import (
	"bufio"
	"errors"
	"net"
	"os"
	"sync"

	"github.com/ngagnon/fly-server/wire"
)

type session struct {
	terminated bool
	user       string
	reader     *bufio.Reader
	out        chan wire.Value
	streams    [16]*stream
	streamLock sync.RWMutex
}

type stream struct {
	finish    chan struct{}
	cancel    chan struct{}
	data      chan []byte
	finalPath string
	file      *os.File
}

var ErrProtocol = errors.New("Protocol error")

func newSession(conn net.Conn) *session {
	return &session{
		terminated: false,
		user:       "",
		reader:     bufio.NewReader(conn),
		out:        make(chan wire.Value, 10),
	}
}

func (s *session) getStream(id int) (stream *stream, ok bool) {
	s.streamLock.RLock()
	defer s.streamLock.RUnlock()

	if id < 0 || id >= len(s.streams) {
		return nil, false
	}

	stream = s.streams[id]
	ok = stream != nil
	return
}

func (s *session) addStream(stream *stream) (id int, ok bool) {
	s.streamLock.Lock()
	defer s.streamLock.Unlock()

	id, ok = nextStreamId(s.streams[:])

	if ok {
		s.streams[id] = stream
	}

	return
}

func (s *session) closeStream(id int) {
	s.streamLock.Lock()
	s.streams[id] = nil
	s.streamLock.Unlock()
}

func nextStreamId(streams []*stream) (id int, ok bool) {
	for i := 0; i < len(streams); i++ {
		if streams[i] == nil {
			return i, true
		}
	}

	return 0, false
}

func (s *session) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}
