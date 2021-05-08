package main

import (
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

func handleAddUser(args []respValue, s *session) respValue {
	if len(args) != 2 {
		return newError("ARG", "Command ADDUSER expects exactly 2 arguments")
	}

	if !checkAdmin(s) {
		return newError("DENIED", "You are not allowed to manage users")
	}

	username, ok := args[0].(*respBlob)

	if !ok {
		return newError("ARG", "Username should be a blob, got %s", args[0].name())
	}

	password, ok := args[1].(*respBlob)

	if !ok {
		return newError("ARG", "Password should be a blob, got %s", args[1].name())
	}

	if len(username.val) < 1 {
		return newError("ARG", "Minimum username length is 1")
	}

	if len(username.val) > 32 {
		return newError("ARG", "Maximum username length is 32")
	}

	if matched, err := regexp.Match("^[a-z_]([a-z0-9_-]{0,31})$", username.val); !matched || err != nil {
		return newError("ARG", "Invalid username")
	}

	hash, err := bcrypt.GenerateFromPassword(password.val, 12)

	if err != nil {
		log.Errorf("Unexpected error while generating hash: %v", err)
		return newError("ERR", "Unexpected error while generating hash")
	}

	updateUsers(func() {
		users[string(username.val)] = user{
			username: string(username.val),
			password: hash,
			chroot:   "",
			admin:    singleUser,
		}

		if singleUser {
			singleUser = false
			s.user = string(username.val)
		}
	})

	return RespOK
}

func handleWhoAmI(args []respValue, s *session) respValue {
	if s.user == "" {
		return &respNull{}
	}

	return &respString{val: s.user}
}

func handleShowUser(args []respValue, s *session) respValue {
	if len(args) != 1 {
		return newError("ARG", "Command SHOWUSER expects exactly 1 argument")
	}

	if !checkAdmin(s) {
		return newError("DENIED", "You are not allowed to manage users.")
	}

	username, ok := args[0].(*respBlob)

	if !ok {
		return newError("ARG", "Username should be a blob, got %s", args[0].name())
	}

	globalLock.RLock()
	defer globalLock.RUnlock()

	user, ok := users[string(username.val)]

	if !ok {
		return newError("NOTFOUND", "User not found")
	}

	result := make(map[string]respValue)
	result["username"] = &respString{val: user.username}
	result["chroot"] = &respString{val: user.chroot}
	result["admin"] = &respBool{val: user.admin}

	return &respMap{m: result}
}

func updateUsers(applyChanges func()) {
	globalLock.Lock()
	defer globalLock.Unlock()

	applyChanges()
	writeUserDb()
}
