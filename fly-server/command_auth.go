package main

import (
	"github.com/ngagnon/fly-server/session"
	"github.com/ngagnon/fly-server/wire"
	"golang.org/x/crypto/bcrypt"
)

func handleAuth(args []wire.Value, s *session.S) wire.Value {
	if len(args) == 0 {
		return wire.NewError("ARG", "Command AUTH expects at least 1 argument")
	}

	authType, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "AUTH type should be a string, got %s", args[0].Name())
	}

	if authType.Value != "PWD" {
		return wire.NewError("ARG", "Unsupported AUTH type: %s", authType.Value)
	}

	if len(args) != 3 {
		return wire.NewError("ARG", "Password authentication requires a username and a password")
	}

	username, ok := args[1].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Username should be a string, got %s", args[1].Name())
	}

	password, ok := args[2].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Password should be a string, got %s", args[2].Name())
	}

	if !verifyPassword(username.Value, password.Value) {
		return wire.NewError("DENIED", "Authentication failed")
	}

	s.SetUser(username.Value)
	return wire.OK
}

func verifyPassword(username string, password string) bool {
	tx := flydb.RTxn()
	user, ok := tx.FindUser(username)
	tx.Complete()

	if !ok {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))

	return err == nil
}
