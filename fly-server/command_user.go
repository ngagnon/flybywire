package main

import (
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

func handleAddUser(args []string, s *session) {
	if len(args) != 2 {
		s.writer.writeError("ERR", "Command ADDUSER expects exactly 2 arguments")
		return
	}

	if !checkAdmin(s) {
		s.writer.writeError("DENIED", "You are not allowed to manage users.")
		return
	}

	username := args[0]
	password := []byte(args[1])

	if len(username) < 1 {
		s.writer.writeError("ERR", "Minimum username length is 1")
		return
	}

	if len(username) > 32 {
		s.writer.writeError("ERR", "Maximum username length is 32")
		return
	}

	if matched, err := regexp.Match("^[a-z_]([a-z0-9_-]{0,31})$", []byte(username)); !matched || err != nil {
		s.writer.writeError("ERR", "Invalid username")
		return
	}

	hash, err := bcrypt.GenerateFromPassword(password, 12)

	if err != nil {
		s.writer.writeError("ERR", "Unexpected error while generating hash")
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

	s.writer.writeOK()
}

func handleWhoAmI(args []string, s *session) {
	if s.user == "" {
		s.writer.writeNull()
	} else {
		s.writer.writeSimpleString(s.user)
	}
}

func handleShowUser(args []string, s *session) {
	if len(args) != 1 {
		s.writer.writeError("ERR", "Command SHOWUSER expects exactly 1 argument")
		return
	}

	if !checkAdmin(s) {
		s.writer.writeError("DENIED", "You are not allowed to manage users.")
		return
	}

	username := args[0]

	globalLock.RLock()
	defer globalLock.RUnlock()

	user, ok := users[username]

	if !ok {
		s.writer.writeError("NOTFOUND", "User not found")
		return
	}

	result := make(map[string]respValue)
	result["username"] = respValue{valueType: RespSimpleString, value: user.username}
	result["chroot"] = respValue{valueType: RespSimpleString, value: user.chroot}
	result["admin"] = respValue{valueType: RespBoolean, value: user.admin}

	s.writer.writeMap(result)
}

func updateUsers(applyChanges func()) {
	globalLock.Lock()
	defer globalLock.Unlock()

	applyChanges()
	writeUserDb()
}
