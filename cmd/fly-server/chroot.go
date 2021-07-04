package main

import (
	"os"

	"github.com/ngagnon/flybywire/internal/db"
	log "github.com/ngagnon/flybywire/internal/logging"
	"github.com/ngagnon/flybywire/internal/vfs"
	"github.com/ngagnon/flybywire/internal/wire"
)

func handleChroot(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 2 {
		return wire.NewError("ARG", "Command CHROOT expects exactly 2 arguments")
	}

	username, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Username should be a string, got %s", args[0].Name())
	}

	chroot, ok := args[1].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Chroot path should be a string, got %s", args[1].Name())
	}

	if s.singleUser {
		return wire.NewError("ILLEGAL", "Cannot manage users in single-user mode")
	}

	if s.user == nil || !s.user.Admin {
		return wire.NewError("DENIED", "You are not allowed to manage users.")
	}

	realPath, err := vfs.ResolveSingleUser(chroot.Value)

	if err != nil {
		return wire.NewError("ARG", "Invalid path")
	}

	if err = os.MkdirAll(realPath, 0755); err != nil {
		log.Debugf("Could not create folder: %v", err)
		return wire.NewError("ERR", "Unexpected error occurred")
	}

	tx := flydb.Txn()
	defer tx.Complete()

	err = tx.UpdateUser(username.Value, func(u *db.User) {
		u.Chroot = chroot.Value
	})

	if err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}

	return wire.OK
}
