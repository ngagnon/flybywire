package session

import (
	"errors"
	"io"
	"os"
	"strconv"
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
	done   chan struct{}
	file   *os.File
}

type writeStream struct {
	frames    chan frame
	cancel    chan struct{}
	done      chan struct{}
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
		return 0, wire.NewError("NOTFOUND", "No such file or directory")
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
		done:   make(chan struct{}),
		file:   file,
	}

	s.streams[id] = stream
	s.streamCount++
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
		done:      make(chan struct{}),
		finalPath: finalPath,
		file:      file,
	}

	s.streams[id] = stream
	s.streamCount++
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

func (s *S) NumStreams() int {
	s.streamLock.RLock()
	defer s.streamLock.RUnlock()
	return s.streamCount
}

func (s *S) releaseStream(id int) {
	s.streamLock.Lock()
	s.streams[id] = nil
	s.streamCount--
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
	defer close(s.done)

	session.waitGroup.Add(1)
	defer session.waitGroup.Done()

	buf := make([]byte, session.ChunkSize)
	tag := strconv.Itoa(id)

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
			session.out <- wire.NewTaggedValue(wire.Null, tag)
			return
		}

		if err != nil {
			err := wire.NewError("IO", "Could not read chunk from file. Closing stream.")
			session.out <- wire.NewTaggedValue(err, tag)
			log.Debugf("Could not read from file: %v", err)
			return
		}

		blob := wire.NewBlob(buf[0:n])
		session.out <- wire.NewTaggedValue(blob, tag)
	}
}

func handleWriteStream(id int, s *writeStream, session *S) {
	defer session.releaseStream(id)
	defer close(s.done)

	session.waitGroup.Add(1)
	defer session.waitGroup.Done()

	watchdog := newWatchdog(1 * time.Minute)
	tag := strconv.Itoa(id)

	for {
		select {
		case <-session.done:
			cancelWriteStream(s)
			return
		case <-s.cancel:
			cancelWriteStream(s)
			return
		case <-watchdog.timeout.C:
			handleTimeout(s, session, tag)
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
			handleTimeout(s, session, tag)
			return
		case frame := <-s.frames:
			if frame.end {
				finishWriteStream(s, tag, session)
				return
			} else {
				ok := handleChunk(frame.payload, tag, s, session, watchdog)

				if !ok {
					return
				}
			}
		}
	}
}

func handleTimeout(s *writeStream, session *S, tag string) {
	cancelWriteStream(s)
	err := wire.NewError("TIMEOUT", "Timed out due to inactivity")
	session.out <- wire.NewTaggedValue(err, tag)
}

func handleChunk(chunk []byte, tag string, s *writeStream, session *S, wd *watchdog) bool {
	_, err := s.file.Write(chunk)

	if err != nil {
		err := wire.NewError("IO", "Could not write chunk to disk. Closing stream.")
		session.out <- wire.NewTaggedValue(err, tag)
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

func finishWriteStream(s *writeStream, tag string, session *S) {
	tmpPath := s.file.Name()
	s.file.Close()

	err := os.Rename(tmpPath, s.finalPath)

	if err != nil {
		err := wire.NewError("IO", "Could not write file to disk.")
		session.out <- wire.NewTaggedValue(err, tag)
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
	<-s.done
}

func (s *readStream) mode() mode {
	return read
}

func (s *readStream) close() {
	s.cancel <- struct{}{}
	<-s.done
}
