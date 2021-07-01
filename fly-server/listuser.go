package main

import "github.com/ngagnon/fly-server/wire"

func handleListUser(args []wire.Value, s *sessionInfo) wire.Value {
	if !checkAdmin(s) {
		return wire.NewError("DENIED", "You are not allowed to manage users")
	}

	if s.singleUser {
		return wire.NewError("ILLEGAL", "Cannot manage users in single-user mode")
	}

	tx := flydb.RTxn()
	users := tx.FetchAllUsers()
	tx.Complete()

	usernames := make([]wire.Value, 0, len(users))

	for _, u := range users {
		usernames = append(usernames, wire.NewString(u.Username))
	}

	return wire.NewArray(usernames)
}
