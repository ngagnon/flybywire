package main

import (
	"github.com/ngagnon/fly-server/wire"
	"golang.org/x/crypto/bcrypt"
)

func handleAuth(args []wire.Value, s *session) wire.Value {
	if len(args) == 0 {
		return wire.NewError("ARG", "Command AUTH expects at least 1 argument")
	}

	authType, ok := args[0].(*wire.Blob)

	if !ok {
		return wire.NewError("ARG", "AUTH type should be a blob, got %s", args[0].Name())
	}

	if string(authType.Data) != "PWD" {
		return wire.NewError("ARG", "Unsupported AUTH type: %s", authType.Data)
	}

	if len(args) != 3 {
		return wire.NewError("ARG", "Password authentication requires a username and a password")
	}

	username, ok := args[1].(*wire.Blob)

	if !ok {
		return wire.NewError("ARG", "Username should be a blob, got %s", args[1].Name())
	}

	password, ok := args[2].(*wire.Blob)

	if !ok {
		return wire.NewError("ARG", "Password should be a blob, got %s", args[2].Name())
	}

	if !verifyPassword(string(username.Data), string(password.Data)) {
		return wire.NewError("DENIED", "Authentication failed")
	}

	s.user = string(username.Data)
	return wire.OK
}

func verifyPassword(username string, password string) bool {
	globalLock.RLock()
	defer globalLock.RUnlock()

	user, ok := users[username]

	if !ok {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.password), []byte(password))

	return err == nil
}
