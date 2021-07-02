package session

import (
	"errors"
	"io"
	"os"
	"path/filepath"
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
	copy
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

type copyStream struct {
	cancel chan struct{}
	done   chan struct{}
	src    string
	dst    string
}

type stream interface {
	close()
	mode() mode
}

func (s *S) NewReadStream(path string) (id int, wirErr *wire.Error) {
	file, err := os.Open(path)

	if errors.Is(err, os.ErrNotExist) {
		return 0, wire.NewError("NOTFOUND", "No such file or directory")
	}

	if err != nil {
		log.Debugf("Could not open file for reading: %v", err)
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

func (s *S) NewWriteStream(finalPath string) (id int, wireErr *wire.Error) {
	parentFolder := filepath.Dir(finalPath)
	info, err := os.Stat(parentFolder)

	if errors.Is(err, os.ErrNotExist) || !info.IsDir() {
		return 0, wire.NewError("NOTFOUND", "No such file or directory")
	}

	file, err := os.CreateTemp("", "flytmp")

	if err != nil {
		log.Debugf("Could not create temporary directory: %v", err)
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

func (s *S) NewCopyStream(src string, dst string) (id int, wireErr *wire.Error) {
	s.streamLock.Lock()
	defer s.streamLock.Unlock()

	id, ok := nextStreamId(s.streams[:])

	if !ok {
		return 0, wire.NewError("TOOMANY", "Too many streams open")
	}

	stream := &copyStream{
		cancel: make(chan struct{}, 2),
		done:   make(chan struct{}),
		src:    src,
		dst:    dst,
	}

	s.streams[id] = stream
	s.streamCount++
	go handleCopyStream(id, stream, s)

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

	tag := strconv.Itoa(id)

	// While one buffer is being written out to the network, we'll start reading into the other buffer
	buf := make([][]byte, 2)
	buf[0] = make([]byte, 32*1024)
	buf[1] = make([]byte, 32*1024)
	cur := 0

	for {
		select {
		case <-session.done:
			return
		case <-s.cancel:
			return
		default:
		}

		cur = (cur + 1) % 2
		n, err := s.file.Read(buf[cur])

		if err == io.EOF {
			session.dataOut <- wire.NewTaggedValue(wire.Null, tag)
			return
		}

		if err != nil {
			log.Debugf("Could not read from file: %v", err)
			wireErr := wire.NewError("IO", "Could not read chunk from file. Closing stream.")
			session.dataOut <- wire.NewTaggedValue(wireErr, tag)
			return
		}

		blob := wire.NewBlob(buf[cur][0:n])
		session.dataOut <- wire.NewTaggedValue(blob, tag)
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

func handleCopyStream(id int, s *copyStream, session *S) {
	defer session.releaseStream(id)
	defer close(s.done)

	session.waitGroup.Add(1)
	defer session.waitGroup.Done()

	tag := strconv.Itoa(id)
	src, err := os.Open(s.src)

	if err != nil {
		log.Debugf("Could not open file: %v", err)
		wireErr := wire.NewError("IO", "Could not open source file. Closing stream.")
		session.dataOut <- wire.NewTaggedValue(wireErr, tag)
		return
	}

	defer src.Close()

	tmp, err := os.CreateTemp("", "flytmp")

	if err != nil {
		log.Debugf("Could not create temporary file: %v", err)
		wireErr := wire.NewError("IO", "Could not create temporary. Closing stream.")
		session.dataOut <- wire.NewTaggedValue(wireErr, tag)
		return
	}

	buf := make([]byte, 32*1024)

	for {
		select {
		case <-session.done:
			cancelCopyStream(tmp)
			return
		case <-s.cancel:
			cancelCopyStream(tmp)
			return
		default:
		}

		written, err := io.CopyBuffer(tmp, io.LimitReader(src, int64(len(buf))), buf)

		if written < int64(len(buf)) && err == nil {
			tmp.Close()

			if err = os.Rename(tmp.Name(), s.dst); err != nil {
				log.Debugf("Could not move temporary file to final destination: %v", err)
				wireErr := wire.NewError("IO", "Could not move temporary file to final destination. Closing stream.")
				session.dataOut <- wire.NewTaggedValue(wireErr, tag)
				return
			}

			session.dataOut <- wire.NewTaggedValue(wire.Null, tag)
			return
		}

		if err != nil {
			log.Debugf("Could not copy chunk: %v", err)
			wireErr := wire.NewError("IO", "Could not copy chunk of data. Closing stream.")
			session.dataOut <- wire.NewTaggedValue(wireErr, tag)
			return
		}
	}
}

func handleTimeout(s *writeStream, session *S, tag string) {
	cancelWriteStream(s)
	err := wire.NewError("TIMEOUT", "Timed out due to inactivity")
	session.dataOut <- wire.NewTaggedValue(err, tag)
}

func handleChunk(chunk []byte, tag string, s *writeStream, session *S, wd *watchdog) bool {
	_, err := s.file.Write(chunk)

	if err != nil {
		log.Debugf("Could not write file to disk: %v", err)
		cancelWriteStream(s)
		wireErr := wire.NewError("IO", "Could not write chunk to disk. Closing stream.")
		session.dataOut <- wire.NewTaggedValue(wireErr, tag)
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
		log.Errorf("Could not write file to disk: %v", err)
		err := wire.NewError("IO", "Could not write file to disk.")
		session.dataOut <- wire.NewTaggedValue(err, tag)
	}
}

func cancelCopyStream(tmp *os.File) {
	tmp.Close()
	os.Remove(tmp.Name())
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

func (s *copyStream) mode() mode {
	return copy
}

func (s *copyStream) close() {
	s.cancel <- struct{}{}
	<-s.done
}
