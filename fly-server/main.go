package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"sync"
)

type commandHandler func([]string, *session) error

var commandHandlers = map[string]commandHandler{
	"PING":     handlePing,
	"QUIT":     handleQuit,
	"MKDIR":    handleMkdir,
	"ADDUSER":  handleAddUser,
	"WHOAMI":   handleWhoAmI,
	"SHOWUSER": handleShowUser,
	"AUTH":     handleAuth,
}

var dir string
var singleUser bool
var users map[string]user
var policies []acp
var globalLock sync.RWMutex

func main() {
	port := flag.Int("port", 6767, "TCP port to listen on")
	flag.Parse()

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))

	if err != nil {
		fmt.Println("Cannot start TCP server:", err)
		os.Exit(1)
	}

	defer ln.Close()

	if flag.NArg() == 0 {
		fmt.Println("USAGE: fly-server ROOTDIR")
		fmt.Println()
		os.Exit(1)
	}

	dir = flag.Arg(0)

	if stat, err := os.Stat(dir); os.IsNotExist(err) || !stat.IsDir() {
		fmt.Println("ERROR: root directory not found:", dir)
		os.Exit(1)
	}

	readDatabase()

	for {
		conn, err := ln.Accept()

		if err != nil {
			fmt.Println("Accept error")
			os.Exit(1)
		}

		go handleSession(conn)
	}
}

func handleSession(conn net.Conn) {
	session := newSession(conn)
	defer conn.Close()

	defer func() {
		if err := recover(); err != nil && err != io.EOF {
			fmt.Println("ERROR: session aborted:", err)
			fmt.Println(string(debug.Stack()))
		}
	}()

	var err error

	for !session.terminated {
		cmd, err := session.nextCommand()
		err = handleProtoError(err, session)

		if err != nil {
			break
		}

		handler, ok := getCommandHandler(cmd.name)

		if ok {
			err = handler(cmd.args, session)
		} else {
			msg := fmt.Sprintf("Unknown command '%s'", cmd.name)
			err = session.writeError("ERR", msg)
		}

		if err != nil {
			break
		}
	}

	if err != nil {
		fmt.Println("ERROR: session aborted:", err)
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
