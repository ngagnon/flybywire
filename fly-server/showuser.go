package main

import "github.com/ngagnon/fly-server/wire"

func handleShowUser(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command SHOWUSER expects exactly 1 argument")
	}

	if !checkAdmin(s) {
		return wire.NewError("DENIED", "You are not allowed to manage users.")
	}

	username, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Username should be a string, got %s", args[0].Name())
	}

	tx := flydb.RTxn()
	user, ok := tx.FindUser(username.Value)
	tx.Complete()

	if !ok {
		return wire.NewError("NOTFOUND", "User not found")
	}

	result := make(map[string]wire.Value)
	result["username"] = wire.NewString(user.Username)
	result["chroot"] = wire.NewString(user.Chroot)
	result["admin"] = wire.NewBoolean(user.Admin)

	return wire.NewMap(result)
}
