package main

import (
	"github.com/ngagnon/fly-server/db"
	log "github.com/ngagnon/fly-server/logging"
	"github.com/ngagnon/fly-server/wire"
	"golang.org/x/crypto/bcrypt"
)

func handleSetpwd(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 2 {
		return wire.NewError("ARG", "Command SETPWD expects exactly 2 arguments")
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

	hash, err := bcrypt.GenerateFromPassword([]byte(password.Value), 12)

	if err != nil {
		log.Errorf("Unexpected error while generating hash: %v", err)
		return wire.NewError("ERR", "Unexpected error while generating hash")
	}

	if s.singleUser {
		return wire.NewError("ILLEGAL", "Cannot manage users in single-user mode")
	}

	if s.username != username.Value && !checkAdmin(s) {
		return wire.NewError("DENIED", "You are not allowed to manage users.")
	}

	tx := flydb.Txn()
	defer tx.Complete()

	err = tx.UpdateUser(username.Value, func(u *db.User) {
		u.Password = hash
	})

	if err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}

	return wire.OK
}
