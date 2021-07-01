package main

import (
	"errors"

	"github.com/ngagnon/fly-server/db"
	log "github.com/ngagnon/fly-server/logging"
	"github.com/ngagnon/fly-server/wire"
)

func handleRmuser(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command RMUSER expects exactly 1 argument")
	}

	if !checkAdmin(s) {
		return wire.NewError("DENIED", "You are not allowed to manage users.")
	}

	username, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Username should be a string, got %s", args[0].Name())
	}

	tx := flydb.Txn()
	err := tx.DeleteUser(username.Value)
	tx.Complete()

	if errors.Is(err, db.ErrNotFound) {
		return wire.NewError("NOTFOUND", "User not found")
	}

	if err != nil {
		log.Fatalf("Failed to delete user: %v", err)
	}

	return wire.OK
}
