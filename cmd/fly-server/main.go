package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/ngagnon/flybywire/internal/crypto"
	"github.com/ngagnon/flybywire/internal/db"
	log "github.com/ngagnon/flybywire/internal/logging"
	"github.com/ngagnon/flybywire/internal/session"
	"github.com/ngagnon/flybywire/internal/vfs"
	"github.com/ngagnon/flybywire/internal/wire"
)

type sessionInfo struct {
	username   string
	user       *db.User
	singleUser bool
	session    *session.S
}

type commandHandler func(args []wire.Value, session *sessionInfo) (response wire.Value)

var commandHandlers = map[string]commandHandler{
	"PING":     handlePing,
	"WHOAMI":   handleWhoAmI,
	"AUTH":     handleAuth,
	"TOKEN":    handleToken,
	"MKDIR":    handleMkdir,
	"TOUCH":    handleTouch,
	"DEL":      handleDel,
	"MOVE":     handleMove,
	"COPY":     handleCopy,
	"LIST":     handleList,
	"LISTUSER": handleListUser,
	"ADDUSER":  handleAddUser,
	"SETPWD":   handleSetpwd,
	"SETADM":   handleSetadm,
	"CHROOT":   handleChroot,
	"RMUSER":   handleRmuser,
	"SHOWUSER": handleShowUser,
	"STREAM":   handleStream,
	"CLOSE":    handleClose,
	"LISTACP":  handleListAcp,
	"PUTACP":   handlePutAcp,
	"RMACP":    handleRmAcp,
}

type policyStore struct{}

var dir string
var flydb *db.Handle
var tokenKey []byte

var (
	port  = flag.Int("port", 6767, "TCP port to listen on")
	notls = flag.Bool("notls", false, "Disable TLS")
	debug = flag.Bool("debug", false, "Turn on debug logging")
)

func main() {
	var err error

	flag.Parse()

	log.Configure(*debug, os.Stderr)

	if flag.NArg() == 0 {
		log.Fatalf("Usage: fly-server ROOTDIR")
	}

	dir = flag.Arg(0)

	if stat, err := os.Stat(dir); os.IsNotExist(err) || !stat.IsDir() {
		log.Fatalf("Root directory not found: %s", dir)
	}

	if flydb, err = db.Open(dir); err != nil {
		log.Fatalf("%v", err)
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))

	if err != nil {
		log.Fatalf("Cannot start TCP server: %v", err)
	}

	if !*notls {
		tlsConfig := &tls.Config{}
		tlsConfig.GetCertificate = flydb.GetCertificate
		ln = tls.NewListener(ln, tlsConfig)
	}

	defer ln.Close()

	vfs.Setup(&policyStore{}, dir)

	tokenKey, err = crypto.RandomKey(16)

	if err != nil {
		log.Errorf("Failed to generate a cryptographic key for token authentication: %v", err)
	}

	log.Infof("Server started. Listening on port %d", *port)

	for {
		conn, err := ln.Accept()

		if err != nil {
			log.Fatalf("Accept error: %v", err)
		}

		s := &sessionInfo{}

		go session.Handle(conn, func(cmd *wire.Array, session *session.S) (response wire.Value) {
			if s.session == nil {
				s.session = session
			}

			return dispatchCommand(cmd, s)
		})
	}
}

func dispatchCommand(cmd *wire.Array, s *sessionInfo) (response wire.Value) {
	cmdName := cmd.Values[0].(*wire.String).Value
	handler, ok := getCommandHandler(cmdName)

	if !ok {
		return wire.NewError("CMD", "Unknown command '%s'", cmdName)
	}

	s.update()

	args := cmd.Values[1:]
	return handler(args, s)
}

func (s *sessionInfo) update() {
	tx := flydb.RTxn()
	defer tx.Complete()

	s.singleUser = tx.NumUsers() == 0

	if s.username == "" {
		return
	}

	// User doesn't exist anymore
	_, ok := tx.FindUser(s.username)

	if !ok {
		s.user = nil
		s.username = ""
	}
}

func (s *sessionInfo) changeUser(username string) {
	tx := flydb.RTxn()
	defer tx.Complete()

	s.username = username
	user, ok := tx.FindUser(s.username)

	if !ok {
		log.Errorf("Tried to change to a non-existing user: %s", username)
		s.session.Terminate()
		return
	}

	s.user = &user
}

func getCommandHandler(s string) (h commandHandler, ok bool) {
	h, ok = commandHandlers[strings.ToUpper(s)]
	return
}

func (s *policyStore) GetPolicies(path string, username string, action db.Action) []db.Policy {
	tx := flydb.RTxn()
	defer tx.Complete()
	return tx.GetPolicies(path, username, action)
}
