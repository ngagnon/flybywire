package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
)

/* @TODO: command handler should return respValue, error */
type commandHandler func([]string, *session) error

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
	defer conn.Close()

	go handleWrites(conn, session.out, session.writeErr)

	var err error

	for !session.terminated {
		if err = checkWriteError(session); err != nil {
			break
		}

		// @TODO: nextCommand -> nextFrame
		/*
			type frame interface {
				streamId *int // nil when not a stream chunk
				val respValue
			}
		*/
		cmd, err := session.nextCommand()
		err = handleProtoError(err, session)

		if err != nil {
			break
		}

		handler, ok := getCommandHandler(cmd.name)

		if ok {
			/* @TODO: handle commands in a separate worker goroutine */
			err = handler(cmd.args, session)
		} else {
			msg := fmt.Sprintf("Unknown command '%s'", cmd.name)
			err = session.writeError("ERR", msg)
		}

		if err != nil {
			break
		}
	}

	/* @TODO: force close all streams (cancel) */
	/* @TODO: drain the out channel */

	if err != nil {
		log.Debugf("Session aborted -- err=\"%v\"", err)
	}
}

func checkWriteError(session *session) (err error) {
	select {
	case err = <-session.writeErr:
	default:
		err = nil
	}

	return
}

func handleWrites(writer io.Writer, out chan respValue, errChan chan error) {
	for {
		val := <-out
		err := val.writeTo(writer)

		if err != nil {
			errChan <- err
			return
		}
	}
}

func handleProtoError(err error, s *session) error {
	if protoErr, ok := err.(*protocolError); ok {
		err = s.writeError("ERR", protoErr.msg)
	}

	return err
}

func getCommandHandler(s string) (h commandHandler, ok bool) {
	h, ok = commandHandlers[strings.ToUpper(s)]
	return
}
