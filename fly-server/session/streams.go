package session

import (
	"os"
	"time"

	log "github.com/ngagnon/fly-server/logging"
	"github.com/ngagnon/fly-server/wire"
)

type frameKind int

const (
	data frameKind = iota
	finish
)

type frame struct {
	end     bool
	payload []byte
}

type stream struct {
	frames    chan frame
	cancel    chan struct{}
	finalPath string
	file      *os.File
}

func (s *S) OpenStream(file *os.File, finalPath string) (id int, ok bool) {
	s.streamLock.Lock()
	defer s.streamLock.Unlock()

	id, ok = nextStreamId(s.streams[:])

	if ok {
		stream := &stream{
			frames:    make(chan frame, 5),
			cancel:    make(chan struct{}, 2),
			finalPath: finalPath,
			file:      file,
		}

		s.streams[id] = stream
		go handleStream(id, stream, s)
	}

	return
}

func (s *S) CloseStream(id int) bool {
	stream, ok := s.getStream(id)

	if !ok {
		return false
	}

	stream.cancel <- struct{}{}

	return true
}

func (s *S) releaseStream(id int) {
	s.streamLock.Lock()
	s.streams[id] = nil
	s.streamLock.Unlock()
}

func (s *S) getStream(id int) (stream *stream, ok bool) {
	s.streamLock.RLock()
	defer s.streamLock.RUnlock()

	if id < 0 || id >= len(s.streams) {
		return nil, false
	}

	stream = s.streams[id]
	ok = stream != nil
	return
}

func nextStreamId(streams []*stream) (id int, ok bool) {
	for i := 0; i < len(streams); i++ {
		if streams[i] == nil {
			return i, true
		}
	}

	return 0, false
}

func handleStream(id int, s *stream, session *S) {
	defer session.releaseStream(id)

	session.waitGroup.Add(1)
	defer session.waitGroup.Done()

	watchdog := newWatchdog(1 * time.Minute)

	for {
		select {
		case <-session.done:
			cancelWriteStream(s)
			return
		case <-s.cancel:
			cancelWriteStream(s)
			return
		case <-watchdog.timeout.C:
			handleTimeout(s, session, id)
			return
		default:
		}

		select {
		case <-session.done:
			cancelWriteStream(s)
			return
		case <-s.cancel:
			cancelWriteStream(s)
			return
		case <-watchdog.timeout.C:
			handleTimeout(s, session, id)
			return
		case frame := <-s.frames:
			if frame.end {
				finishWriteStream(s, id, session)
				return
			} else {
				ok := handleChunk(frame.payload, id, s, session, watchdog)

				if !ok {
					return
				}
			}
		}
	}
}

func handleTimeout(s *stream, session *S, id int) {
	cancelWriteStream(s)
	err := wire.NewError("TIMEOUT", "Timed out due to inactivity")
	session.out <- wire.NewStreamFrame(id, err)
}

func handleChunk(chunk []byte, id int, s *stream, session *S, wd *watchdog) bool {
	_, err := s.file.Write(chunk)

	if err != nil {
		err := wire.NewError("IO", "Could not write chunk to disk. Closing stream.")
		session.out <- wire.NewStreamFrame(id, err)
		log.Debugf("Could not write file to disk: %v", err)
		cancelWriteStream(s)
		return false
	}

	wd.reset()

	return true
}

func cancelWriteStream(s *stream) {
	s.file.Close()
	os.Remove(s.file.Name())
}

func finishWriteStream(s *stream, id int, session *S) {
	tmpPath := s.file.Name()
	s.file.Close()

	err := os.Rename(tmpPath, s.finalPath)

	if err != nil {
		err := wire.NewError("IO", "Could not write file to disk.")
		session.out <- wire.NewStreamFrame(id, err)
		log.Errorf("Could not write file to disk: %v", err)
	}
}

func newDataFrame(payload []byte) frame {
	return frame{end: false, payload: payload}
}

func newFinishFrame() frame {
	return frame{end: true}
}
