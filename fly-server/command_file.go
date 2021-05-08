package main

import (
	"os"
	"strings"
	"time"
)

func handleMkdir(args []respValue, s *session) respValue {
	if len(args) != 1 {
		return newError("ARG", "Command MKDIR expects exactly one argument")
	}

	pathBlob, ok := args[0].(*respBlob)

	if !ok {
		return newError("ARG", "Path should be a blob, got %s", args[0].name())
	}

	vPath := "/" + strings.Trim(string(pathBlob.val), "/")

	if !checkAuth(s, vPath, true) {
		return newError("DENIED", "Access denied")
	}

	realPath := resolveVirtualPath(vPath)

	if err := os.MkdirAll(realPath, 0755); err != nil {
		// @TODO: debug log
		return newError("ERR", "Unexpected error occurred")
	}

	return RespOK
}

func handleStream(args []respValue, s *session) respValue {
	if len(args) != 2 {
		return newError("ARG", "Command STREAM expects exactly 2 arguments")
	}

	mode, ok := args[0].(*respBlob)

	if !ok {
		return newError("ARG", "Mode should be a blob, got %s", args[0].name())
	}

	pathBlob, ok := args[0].(*respBlob)

	if !ok {
		return newError("ARG", "Path should be a blob, got %s", args[0].name())
	}

	vPath := "/" + strings.TrimPrefix(string(pathBlob.val), "/")

	if string(mode.val) != "W" {
		return newError("ARG", "Unsupported mode: %s", mode.val)
	}

	if !checkAuth(s, vPath, true) {
		return newError("DENIED", "Access denied")
	}

	/* @TODO: check that the folder exists */

	f, err := os.CreateTemp("", "flytmp")

	if err != nil {
		// @TODO: debug log
		return newError("ERR", "Unexpected error occurred")
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
		return newError("TOOMANY", "Too many streams open")
	}

	go handleWriteStream(id, stream, s)

	return &respInteger{val: id}
}

func handleWriteStream(id int, s *stream, session *session) {
	defer session.closeStream(id)

	maxInactivity := 1 * time.Minute
	timeout := time.NewTimer(maxInactivity)

	for {
		select {
		case <-timeout.C:
			cancelWriteStream(s)
			session.out <- &respError{code: "TIMEOUT", msg: "Timed out due to inactivity"}
			return
		case <-s.cancel:
			cancelWriteStream(s)
			return
		case <-s.finish:
			err := finishWriteStream(s)

			if err != nil {
				session.out <- &respError{code: "IO", msg: "Could not write file to disk."}
				log.Debugf("Could not write file to disk -- err=\"%v\"", err)
			}

			return
		case chunk := <-s.data:
			_, err := s.file.Write(chunk)

			if err != nil {
				session.out <- &respError{code: "IO", msg: "Could not write chunk to disk. Closing stream."}
				log.Debugf("Could not write file to disk -- err=\"%v\"", err)
				cancelWriteStream(s)
				return
			} else {
				if !timeout.Stop() {
					<-timeout.C
				}

				timeout.Reset(maxInactivity)
			}
		}
	}
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
