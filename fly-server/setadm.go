package main

import (
	"github.com/ngagnon/fly-server/db"
	log "github.com/ngagnon/fly-server/logging"
	"github.com/ngagnon/fly-server/wire"
)

func handleSetadm(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 2 {
		return wire.NewError("ARG", "Command SETADM expects exactly 2 arguments")
	}

	username, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Username should be a string, got %s", args[0].Name())
	}

	admin, ok := args[1].(*wire.Bool)

	if !ok {
		return wire.NewError("ARG", "Admin bit should be a boolean, got %s", args[1].Name())
	}

	if s.singleUser {
		return wire.NewError("ILLEGAL", "Cannot manage users in single-user mode")
	}

	if s.user == nil || !s.user.Admin {
		return wire.NewError("DENIED", "You are not allowed to manage users.")
	}

	tx := flydb.Txn()
	defer tx.Complete()

	err := tx.UpdateUser(username.Value, func(u *db.User) {
		u.Admin = admin.Value
	})

	if err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}

	return wire.OK
}
