package main

import (
	"errors"

	"github.com/ngagnon/flybywire/internal/db"
	log "github.com/ngagnon/flybywire/internal/logging"
	"github.com/ngagnon/flybywire/internal/wire"
	"golang.org/x/crypto/bcrypt"
)

func handleAddUser(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 2 {
		return wire.NewError("ARG", "Command ADDUSER expects exactly 2 arguments")
	}

	if !s.singleUser && (s.user == nil || !s.user.Admin) {
		return wire.NewError("DENIED", "You are not allowed to manage users")
	}

	username, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Username should be a string, got %s", args[0].Name())
	}

	password, ok := args[1].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Password should be a string, got %s", args[1].Name())
	}

	if len(password.Value) == 0 {
		return wire.NewError("ARG", "Password cannot be empty")
	}

	if len(username.Value) < 1 {
		return wire.NewError("ARG", "Minimum username length is 1")
	}

	if len(username.Value) > 32 {
		return wire.NewError("ARG", "Maximum username length is 32")
	}

	if !db.ValidateUsername(username.Value) {
		return wire.NewError("ARG", "Invalid username")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password.Value), 12)

	if err != nil {
		log.Errorf("Unexpected error while generating hash: %v", err)
		return wire.NewError("ERR", "Unexpected error while generating hash")
	}

	newUser := &db.User{
		Username: username.Value,
		Password: hash,
		Chroot:   "",
		Admin:    s.singleUser,
	}

	tx := flydb.Txn()
	err = tx.AddUser(newUser)
	tx.Complete()

	if errors.Is(err, db.ErrExists) {
		return wire.NewError("EXISTS", "Already exists")
	}

	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	if s.singleUser {
		s.changeUser(newUser.Username)
	}

	return wire.OK
}
