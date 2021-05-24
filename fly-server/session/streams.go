package session

import (
	"errors"
	"io"
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

type mode int

const (
	read mode = iota
	write
)

type frame struct {
	end     bool
	payload []byte
}

type readStream struct {
	cancel chan struct{}
	file   *os.File
}

type writeStream struct {
	frames    chan frame
	cancel    chan struct{}
	finalPath string
	file      *os.File
}

type stream interface {
	close()
	mode() mode
}

/* @TODO: check that the file exists */
func (s *S) NewReadStream(path string) (id int, wirErr *wire.Error) {
	file, err := os.Open(path)

	if errors.Is(err, os.ErrNotExist) {
		return 0, wire.NewError("NOTFOUND", "File not found")
	}

	if err != nil {
		// @TODO: debug log
		return 0, wire.NewError("ERR", "Unexpected error occurred")
	}

	s.streamLock.Lock()
	defer s.streamLock.Unlock()

	id, ok := nextStreamId(s.streams[:])

	if !ok {
		file.Close()
		return 0, wire.NewError("TOOMANY", "Too many streams open")
	}

	stream := &readStream{
		cancel: make(chan struct{}, 2),
		file:   file,
	}

	s.streams[id] = stream
	go handleReadStream(id, stream, s)

	return id, nil
}

/* @TODO: check that the parent folder of finalPath exists */
func (s *S) NewWriteStream(finalPath string) (id int, wireErr *wire.Error) {
	file, err := os.CreateTemp("", "flytmp")

	if err != nil {
		// @TODO: debug log
		return 0, wire.NewError("ERR", "Unexpected error occurred")
	}

	s.streamLock.Lock()
	defer s.streamLock.Unlock()

	id, ok := nextStreamId(s.streams[:])

	if !ok {
		file.Close()
		return 0, wire.NewError("TOOMANY", "Too many streams open")
	}

	stream := &writeStream{
		frames:    make(chan frame, 5),
		cancel:    make(chan struct{}, 2),
		finalPath: finalPath,
		file:      file,
	}

	s.streams[id] = stream
	go handleWriteStream(id, stream, s)

	return id, nil
}

func (s *S) CloseStream(id int) bool {
	stream, ok := s.getStream(id)

	if !ok {
		return false
	}

	stream.close()

	return true
}

func (s *S) releaseStream(id int) {
	s.streamLock.Lock()
	s.streams[id] = nil
	s.streamLock.Unlock()
}

func (s *S) getStream(id int) (stream stream, ok bool) {
	s.streamLock.RLock()
	defer s.streamLock.RUnlock()

	if id < 0 || id >= len(s.streams) {
		return nil, false
	}

	stream = s.streams[id]
	ok = stream != nil
	return
}

func nextStreamId(streams []stream) (id int, ok bool) {
	for i := 0; i < len(streams); i++ {
		if streams[i] == nil {
			return i, true
		}
	}

	return 0, false
}

func handleReadStream(id int, s *readStream, session *S) {
	defer session.releaseStream(id)
	defer s.file.Close()

	session.waitGroup.Add(1)
	defer session.waitGroup.Done()

	// @TODO: use max chunk size sent by client
	chunkSize := 16 * 1024
	buf := make([]byte, chunkSize)

	for {
		select {
		case <-session.done:
			return
		case <-s.cancel:
			return
		default:
		}

		n, err := s.file.Read(buf)

		if err == io.EOF {
			session.out <- wire.NewStreamFrame(id, wire.Null)
			return
		}

		if err != nil {
			err := wire.NewError("IO", "Could not read chunk from file. Closing stream.")
			session.out <- wire.NewStreamFrame(id, err)
			log.Debugf("Could not read from file: %v", err)
			return
		}

		blob := wire.NewBlob(buf[0:n])
		session.out <- wire.NewStreamFrame(id, blob)
	}
}

func handleWriteStream(id int, s *writeStream, session *S) {
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

func handleTimeout(s *writeStream, session *S, id int) {
	cancelWriteStream(s)
	err := wire.NewError("TIMEOUT", "Timed out due to inactivity")
	session.out <- wire.NewStreamFrame(id, err)
}

func handleChunk(chunk []byte, id int, s *writeStream, session *S, wd *watchdog) bool {
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

func cancelWriteStream(s *writeStream) {
	s.file.Close()
	os.Remove(s.file.Name())
}

func finishWriteStream(s *writeStream, id int, session *S) {
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

func (s *writeStream) mode() mode {
	return write
}

func (s *writeStream) close() {
	s.cancel <- struct{}{}
}

func (s *readStream) mode() mode {
	return read
}

func (s *readStream) close() {
	s.cancel <- struct{}{}
}
