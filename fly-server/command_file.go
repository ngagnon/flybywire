package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func handleMkdir(args []string, s *session) error {
	if len(args) != 1 {
		return s.writeError("ERR", "Command MKDIR expects exactly one argument")
	}

	vPath := "/" + strings.Trim(args[0], "/")

	if !checkAuth(s, vPath, true) {
		return s.writeError("DENIED", "Access denied")
	}

	realPath := resolveVirtualPath(vPath)

	if err := os.MkdirAll(realPath, 0755); err != nil {
		// @TODO: write an error to the session (unexpected error)
		return err
	}

	return s.writeOK()
}

func handleStream(args []string, s *session) error {
	if len(args) != 2 {
		return s.writeError("ERR", "Command STREAM expects exactly 2 arguments")
	}

	mode := args[0]
	vPath := "/" + strings.TrimPrefix(args[1], "/")

	if mode != "W" {
		msg := fmt.Sprint("Unsupported mode:", mode)
		return s.writeError("ERR", msg)
	}

	if !checkAuth(s, vPath, true) {
		return s.writeError("DENIED", "Access denied")
	}

	/* @TODO: check that the folder exists */

	f, err := os.CreateTemp("", "flytmp")

	if err != nil {
		// @TODO: write an error to the session (unexpected error)
		return err
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
		return s.writeError("TOOMANY", "Too many streams open")
	}

	go handleWriteStream(id, stream, s)

	return s.writeInt(id)
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
