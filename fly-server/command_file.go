package main

import (
	"os"
	"strings"
	"time"

	"github.com/ngagnon/fly-server/wire"
)

func handleMkdir(args []wire.Value, s *session) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command MKDIR expects exactly one argument")
	}

	pathBlob, ok := args[0].(*wire.Blob)

	if !ok {
		return wire.NewError("ARG", "Path should be a blob, got %s", args[0].Name())
	}

	vPath := "/" + strings.Trim(string(pathBlob.Data), "/")

	if !checkAuth(s, vPath, true) {
		return wire.NewError("DENIED", "Access denied")
	}

	realPath := resolveVirtualPath(vPath)

	if err := os.MkdirAll(realPath, 0755); err != nil {
		// @TODO: debug log
		return wire.NewError("ERR", "Unexpected error occurred")
	}

	return wire.OK
}

func handleStream(args []wire.Value, s *session) wire.Value {
	if len(args) != 2 {
		return wire.NewError("ARG", "Command STREAM expects exactly 2 arguments")
	}

	mode, ok := args[0].(*wire.Blob)

	if !ok {
		return wire.NewError("ARG", "Mode should be a blob, got %s", args[0].Name())
	}

	pathBlob, ok := args[1].(*wire.Blob)

	if !ok {
		return wire.NewError("ARG", "Path should be a blob, got %s", args[1].Name())
	}

	vPath := "/" + strings.TrimPrefix(string(pathBlob.Data), "/")

	if string(mode.Data) != "W" {
		return wire.NewError("ARG", "Unsupported mode: %s", mode.Data)
	}

	if !checkAuth(s, vPath, true) {
		return wire.NewError("DENIED", "Access denied")
	}

	/* @TODO: check that the folder exists */

	f, err := os.CreateTemp("", "flytmp")

	if err != nil {
		// @TODO: debug log
		return wire.NewError("ERR", "Unexpected error occurred")
	}

	realPath := resolveVirtualPath(vPath)

	stream := &stream{
		finish:    make(chan struct{}, 2),
		cancel:    make(chan struct{}, 2),
		data:      make(chan []byte, 5),
		finalPath: realPath,
		file:      f,
	}

	id, ok := s.addStream(stream)

	if !ok {
		f.Close()
		return wire.NewError("TOOMANY", "Too many streams open")
	}

	go handleWriteStream(id, stream, s)

	return wire.NewInteger(id)
}

func handleWriteStream(id int, s *stream, session *session) {
	defer session.closeStream(id)

	maxInactivity := 1 * time.Minute
	timeout := time.NewTimer(maxInactivity)

	for {
		select {
		case chunk := <-s.data:
			ok := handleChunk(chunk, s, session, timeout, maxInactivity)

			if !ok {
				return
			}

			continue
		default:
		}

		select {
		case chunk := <-s.data:
			ok := handleChunk(chunk, s, session, timeout, maxInactivity)

			if !ok {
				return
			}
		case <-timeout.C:
			cancelWriteStream(s)
			session.out <- wire.NewError("TIMEOUT", "Timed out due to inactivity")
			return
		case <-s.cancel:
			cancelWriteStream(s)
			return
		case <-s.finish:
			err := finishWriteStream(s)

			if err != nil {
				session.out <- wire.NewError("IO", "Could not write file to disk.")
				log.Debugf("Could not write file to disk -- err=\"%v\"", err)
			}

			return
		}
	}
}

func handleChunk(chunk []byte, s *stream, session *session, timeout *time.Timer, maxInactivity time.Duration) (ok bool) {
	_, err := s.file.Write(chunk)

	if err != nil {
		session.out <- wire.NewError("IO", "Could not write chunk to disk. Closing stream.")
		log.Debugf("Could not write file to disk -- err=\"%v\"", err)
		cancelWriteStream(s)
		return false
	}

	// @TODO: refactor into a watchdog timer
	if !timeout.Stop() {
		<-timeout.C
	}

	timeout.Reset(maxInactivity)

	return true
}

func cancelWriteStream(s *stream) {
	s.file.Close()
	os.Remove(s.file.Name())
}

func finishWriteStream(s *stream) error {
	tmpPath := s.file.Name()
	s.file.Close()
	return os.Rename(tmpPath, s.finalPath)
}
