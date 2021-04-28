package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func handleAuth(args []string, s *session) error {
	if len(args) == 0 {
		return s.writeError("ERR", "Command AUTH expects at least 1 argument")
	}

	authType := args[0]

	if authType != "PWD" {
		msg := fmt.Sprint("Unsupported AUTH type:", authType)
		return s.writeError("ERR", msg)
	}

	if len(args) != 3 {
		return s.writeError("ERR", "Password authentication requires a username and a password")
	}

	username := args[1]
	password := args[2]

	if verifyPassword(username, password) {
		s.user = username
		return s.writeOK()
	} else {
		return s.writeError("DENIED", "Authentication failed")
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
