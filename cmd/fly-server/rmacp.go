package main

import (
	"errors"

	"github.com/ngagnon/flybywire/internal/db"
	log "github.com/ngagnon/flybywire/internal/logging"
	"github.com/ngagnon/flybywire/internal/wire"
)

func handleRmAcp(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command RMACP expects exactly 1 argument")
	}

	if s.singleUser {
		return wire.NewError("ILLEGAL", "Cannot manage ACPs in single-user mode")
	}

	if s.user == nil || !s.user.Admin {
		return wire.NewError("DENIED", "You are not allowed to manage users")
	}

	name, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Name should be a string, got %s", args[0].Name())
	}

	tx := flydb.Txn()
	err := tx.DeleteAccessPolicy(name.Value)
	tx.Complete()

	if errors.Is(err, db.ErrNotFound) {
		return wire.NewError("NOTFOUND", "Policy not found")
	}

	if err != nil {
		log.Fatalf("Failed to delete policy: %v", err)
	}

	return wire.OK
}
