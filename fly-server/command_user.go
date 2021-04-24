package main

import (
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

func handleAddUser(args []string, s *session) {
	if len(args) != 2 {
		msg := "-ERR Command ADDUSER expects exactly 2 arguments\r\n"
		s.writer.Write([]byte(msg))
		return
	}

	username := args[0]
	password := []byte(args[1])

	if len(username) < 1 {
		s.writer.Write([]byte("+ERR Minimum username length is 1\r\n"))
		return
	}

	if len(username) > 32 {
		s.writer.Write([]byte("+ERR Maximum username length is 32\r\n"))
		return
	}

	if matched, err := regexp.Match("^[a-z_]([a-z0-9_-]{0,31})$", []byte(username)); !matched || err != nil {
		s.writer.Write([]byte("+ERR Invalid username\r\n"))
		return
	}

	hash, err := bcrypt.GenerateFromPassword(password, 12)

	if err != nil {
		s.writer.Write([]byte("+ERR Unexpected error while generating hash\r\n"))
		return
	}

	updateUsers(func() {
		users[username] = user{
			username: username,
			password: hash,
			chroot:   "",
			admin:    singleUser,
		}

		if singleUser {
			singleUser = false
			s.user = username
		}
	})

	s.writer.Write([]byte("+OK\r\n"))
}

func updateUsers(applyChanges func()) {
	globalLock.Lock()
	defer globalLock.Unlock()

	applyChanges()
	writeUserDb()
}
