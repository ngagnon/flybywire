package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/ngagnon/fly-server/db"
	log "github.com/ngagnon/fly-server/logging"
	"github.com/ngagnon/fly-server/session"
	"github.com/ngagnon/fly-server/wire"
)

type commandHandler func(args []wire.Value, session *session.S) (response wire.Value)

var commandHandlers = map[string]commandHandler{
	"PING":     handlePing,
	"WHOAMI":   handleWhoAmI,
	"AUTH":     handleAuth,
	"MKDIR":    handleMkdir,
	"ADDUSER":  handleAddUser,
	"SHOWUSER": handleShowUser,
	"STREAM":   handleStream,
	"CLOSE":    handleClose,
}

var dir string
var flydb *db.Handle

func main() {
	port := flag.Int("port", 6767, "TCP port to listen on")
	debug := flag.Bool("debug", false, "Turn on debug logging")
	flag.Parse()

	log.Configure(*debug, os.Stderr)

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

	if flydb, err = db.Open(dir); err != nil {
		log.Fatalf("%v", err)
	}

	log.Infof("Server started. Listening on port %d", *port)

	for {
		conn, err := ln.Accept()

		if err != nil {
			log.Fatalf("Accept error: %v", err)
		}

		go session.Handle(conn, dispatchCommand)
	}
}

func dispatchCommand(cmd *wire.Array, session *session.S) (response wire.Value) {
	cmdName := string(cmd.Values[0].(*wire.Blob).Data)
	handler, ok := getCommandHandler(cmdName)

	if !ok {
		return wire.NewError("CMD", "Unknown command '%s'", cmdName)
	}

	args := cmd.Values[1:]
	return handler(args, session)
}

func getCommandHandler(s string) (h commandHandler, ok bool) {
	h, ok = commandHandlers[strings.ToUpper(s)]
	return
}
