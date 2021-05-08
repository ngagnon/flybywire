package main

import (
	"golang.org/x/crypto/bcrypt"
)

func handleAuth(args []respValue, s *session) respValue {
	if len(args) == 0 {
		return newError("ARG", "Command AUTH expects at least 1 argument")
	}

	authType, ok := args[0].(*respBlob)

	if !ok {
		return newError("ARG", "AUTH type should be a blob, got %s", args[0].name())
	}

	if string(authType.val) != "PWD" {
		return newError("ARG", "Unsupported AUTH type: %s", authType.val)
	}

	if len(args) != 3 {
		return newError("ARG", "Password authentication requires a username and a password")
	}

	username, ok := args[1].(*respBlob)

	if !ok {
		return newError("ARG", "Username should be a blob, got %s", args[1].name())
	}

	password, ok := args[2].(*respBlob)

	if !ok {
		return newError("ARG", "Password should be a blob, got %s", args[2].name())
	}

	if !verifyPassword(string(username.val), string(password.val)) {
		return newError("DENIED", "Authentication failed")
	}

	s.user = string(username.val)
	return RespOK
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
