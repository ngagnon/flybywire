package main

import (
	"regexp"

	"github.com/ngagnon/fly-server/wire"
	"golang.org/x/crypto/bcrypt"
)

func handleAddUser(args []wire.Value, s *session) wire.Value {
	if len(args) != 2 {
		return wire.NewError("ARG", "Command ADDUSER expects exactly 2 arguments")
	}

	if !checkAdmin(s) {
		return wire.NewError("DENIED", "You are not allowed to manage users")
	}

	username, ok := args[0].(*wire.Blob)

	if !ok {
		return wire.NewError("ARG", "Username should be a blob, got %s", args[0].Name())
	}

	password, ok := args[1].(*wire.Blob)

	if !ok {
		return wire.NewError("ARG", "Password should be a blob, got %s", args[1].Name())
	}

	if len(username.Data) < 1 {
		return wire.NewError("ARG", "Minimum username length is 1")
	}

	if len(username.Data) > 32 {
		return wire.NewError("ARG", "Maximum username length is 32")
	}

	if matched, err := regexp.Match("^[a-z_]([a-z0-9_-]{0,31})$", username.Data); !matched || err != nil {
		return wire.NewError("ARG", "Invalid username")
	}

	hash, err := bcrypt.GenerateFromPassword(password.Data, 12)

	if err != nil {
		log.Errorf("Unexpected error while generating hash: %v", err)
		return wire.NewError("ERR", "Unexpected error while generating hash")
	}

	updateUsers(func() {
		users[string(username.Data)] = user{
			username: string(username.Data),
			password: hash,
			chroot:   "",
			admin:    singleUser,
		}

		if singleUser {
			singleUser = false
			s.user = string(username.Data)
		}
	})

	return wire.OK
}

func handleWhoAmI(args []wire.Value, s *session) wire.Value {
	if s.user == "" {
		return wire.Null
	}

	return wire.NewString(s.user)
}

func handleShowUser(args []wire.Value, s *session) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command SHOWUSER expects exactly 1 argument")
	}

	if !checkAdmin(s) {
		return wire.NewError("DENIED", "You are not allowed to manage users.")
	}

	username, ok := args[0].(*wire.Blob)

	if !ok {
		return wire.NewError("ARG", "Username should be a blob, got %s", args[0].Name())
	}

	globalLock.RLock()
	defer globalLock.RUnlock()

	user, ok := users[string(username.Data)]

	if !ok {
		return wire.NewError("NOTFOUND", "User not found")
	}

	result := make(map[string]wire.Value)
	result["username"] = wire.NewString(user.username)
	result["chroot"] = wire.NewString(user.chroot)
	result["admin"] = wire.NewBoolean(user.admin)

	return wire.NewMap(result)
}

func updateUsers(applyChanges func()) {
	globalLock.Lock()
	defer globalLock.Unlock()

	applyChanges()
	writeUserDb()
}
