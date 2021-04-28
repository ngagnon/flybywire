package main

import (
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

func handleAddUser(args []string, s *session) error {
	if len(args) != 2 {
		return s.writeError("ERR", "Command ADDUSER expects exactly 2 arguments")
	}

	if !checkAdmin(s) {
		return s.writeError("DENIED", "You are not allowed to manage users.")
	}

	username := args[0]
	password := []byte(args[1])

	if len(username) < 1 {
		return s.writeError("ERR", "Minimum username length is 1")
	}

	if len(username) > 32 {
		return s.writeError("ERR", "Maximum username length is 32")
	}

	if matched, err := regexp.Match("^[a-z_]([a-z0-9_-]{0,31})$", []byte(username)); !matched || err != nil {
		return s.writeError("ERR", "Invalid username")
	}

	hash, err := bcrypt.GenerateFromPassword(password, 12)

	if err != nil {
		return s.writeError("ERR", "Unexpected error while generating hash")
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

	return s.writeOK()
}

func handleWhoAmI(args []string, s *session) error {
	if s.user == "" {
		return s.writeNull()
	}

	return s.writeSimpleString(s.user)
}

func handleShowUser(args []string, s *session) error {
	if len(args) != 1 {
		return s.writeError("ERR", "Command SHOWUSER expects exactly 1 argument")
	}

	if !checkAdmin(s) {
		return s.writeError("DENIED", "You are not allowed to manage users.")
	}

	username := args[0]

	globalLock.RLock()
	defer globalLock.RUnlock()

	user, ok := users[username]

	if !ok {
		return s.writeError("NOTFOUND", "User not found")
	}

	result := make(map[string]respValue)
	result["username"] = respValue{valueType: RespSimpleString, value: user.username}
	result["chroot"] = respValue{valueType: RespSimpleString, value: user.chroot}
	result["admin"] = respValue{valueType: RespBoolean, value: user.admin}

	return s.writeMap(result)
}

func updateUsers(applyChanges func()) {
	globalLock.Lock()
	defer globalLock.Unlock()

	applyChanges()
	writeUserDb()
}
