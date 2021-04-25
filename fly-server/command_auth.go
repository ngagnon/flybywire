package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func handleAuth(args []string, s *session) {
	if len(args) == 0 {
		s.writer.writeError("ERR", "Command AUTH expects at least 1 argument")
		return
	}

	authType := args[0]

	if authType != "PWD" {
		msg := fmt.Sprint("Unsupported AUTH type:", authType)
		s.writer.writeError("ERR", msg)
		return
	}

	if len(args) != 3 {
		s.writer.writeError("ERR", "Password authentication requires a username and a password")
		return
	}

	username := args[1]
	password := args[2]

	if verifyPassword(username, password) {
		s.user = username
		s.writer.writeOK()
	} else {
		s.writer.writeError("DENIED", "Authentication failed")
	}
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
