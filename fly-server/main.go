package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/ngagnon/fly-server/wire"
)

type commandHandler func(args []wire.Value, session *session) (response wire.Value)

var commandHandlers = map[string]commandHandler{
	"PING":     handlePing,
	"QUIT":     handleQuit,
	"WHOAMI":   handleWhoAmI,
	"AUTH":     handleAuth,
	"MKDIR":    handleMkdir,
	"ADDUSER":  handleAddUser,
	"SHOWUSER": handleShowUser,
	"STREAM":   handleStream,
}

var dir string
var singleUser bool
var users map[string]user
var policies []acp
var globalLock sync.RWMutex

func main() {
	port := flag.Int("port", 6767, "TCP port to listen on")
	debug := flag.Bool("debug", false, "Turn on debug logs")
	flag.Parse()

	log.Init(*debug, os.Stderr)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))

	if err != nil {
		log.Fatalf("Cannot start TCP server: %v", err)
	}

	defer ln.Close()

	if flag.NArg() == 0 {
		log.Fatalf("Usage: fly-server ROOTDIR", err)
	}

	dir = flag.Arg(0)

	if stat, err := os.Stat(dir); os.IsNotExist(err) || !stat.IsDir() {
		log.Fatalf("Root directory not found: %s", dir)
	}

	readDatabase()

	log.Infof("Server started. Listening on port %d", *port)

	for {
		conn, err := ln.Accept()

		if err != nil {
			log.Fatalf("Accept error: %v", err)
		}

		go handleSession(conn)
	}
}

func handleSession(conn net.Conn) {
	session := newSession(conn)
	writeErr := make(chan error)
	writerDone := false
	defer conn.Close()

	go handleWrites(conn, session.out, writeErr)

	var err error

	for !session.terminated {
		if err = checkWriteError(writeErr); err != nil {
			writerDone = true
			break
		}

		frame, err := wire.ReadFrame(session.reader)

		if errors.Is(err, wire.ErrFormat) {
			session.out <- wire.NewError("PROTO", err.Error())
			continue
		}

		if err != nil {
			break
		}

		if frame.StreamId == nil {
			arr := frame.Payload.(*wire.Array)
			cmdName := string(arr.Values[0].(*wire.Blob).Data)
			handler, ok := getCommandHandler(cmdName)

			if ok {
				/* @TODO: handle commands in a separate worker goroutine */
				args := arr.Values[1:]
				response := handler(args, session)
				session.out <- response
			} else {
				session.out <- wire.NewError("CMD", "Unknown command '%s'", cmdName)
			}
		} else {
			stream, ok := session.getStream(*frame.StreamId)

			if !ok {
				session.out <- wire.NewError("PROTO", "Invalid stream ID %d", *frame.StreamId)
				continue
			}

			if blob, isBlob := frame.Payload.(*wire.Blob); isBlob {
				stream.data <- blob.Data
			} else if frame.Payload == wire.Null {
				stream.finish <- struct{}{}
			} else {
				session.out <- wire.NewError("PROTO", "Expected blob or null after stream header, got %s", frame.Payload.Name())
			}
		}
	}

	/* @TODO: force close all streams (cancel) */

	if !writerDone {
		// Put an "end-of-file" marker in write queue
		session.out <- nil

		// Wait for the writer to drain the write queue (or fail on error)
		<-writeErr
	}
}

func checkWriteError(writeErr chan error) (err error) {
	select {
	case err = <-writeErr:
	default:
		err = nil
	}

	return
}

/* @TODO: move to its own struct/file */
func handleWrites(writer io.Writer, out chan wire.Value, errChan chan error) {
	for {
		val := <-out

		if val == nil {
			errChan <- nil
			return
		}

		err := val.WriteTo(writer)

		if err != nil {
			errChan <- err
			return
		}
	}
}

func getCommandHandler(s string) (h commandHandler, ok bool) {
	h, ok = commandHandlers[strings.ToUpper(s)]
	return
}
