package main

import (
	"strings"

	"github.com/ngagnon/fly-server/db"
	log "github.com/ngagnon/fly-server/logging"
	"github.com/ngagnon/fly-server/wire"
)

func handlePutAcp(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 5 {
		return wire.NewError("ARG", "Command PUTACP expects exactly 5 arguments")
	}

	if s.singleUser {
		return wire.NewError("ILLEGAL", "Cannot manage ACPs in single-user mode")
	}

	if !checkAdmin(s) {
		return wire.NewError("DENIED", "You are not allowed to manage users")
	}

	name, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Name should be a string, got %s", args[0].Name())
	}

	if len(strings.TrimSpace(name.Value)) == 0 {
		return wire.NewError("ARG", "Name is missing")
	}

	verb, ok := args[1].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Verb should be a string, got %s", args[1].Name())
	}

	if verb.Value != "ALLOW" && verb.Value != "DENY" {
		return wire.NewError("ARG", "Verb should be ALLOW or DENY, got %s", verb.Value)
	}

	action, ok := args[2].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Action should be a string, got %s", args[2].Name())
	}

	if action.Value != "R" && action.Value != "W" {
		return wire.NewError("ARG", "Action should be R or W, got %s", action.Value)
	}

	users, ok := args[3].(*wire.Array)

	if !ok {
		return wire.NewError("ARG", "User paramater should be an array, got %s", args[3].Name())
	}

	if len(users.Values) == 0 {
		return wire.NewError("ARG", "At least one user must be specified")
	}

	paths, ok := args[4].(*wire.Array)

	if !ok {
		return wire.NewError("ARG", "Path paramater should be an array, got %s", args[4].Name())
	}

	if len(paths.Values) == 0 {
		return wire.NewError("ARG", "At least one path must be specified")
	}

	policy := &db.Policy{
		Name:   name.Value,
		Verb:   db.Verb(verb.Value),
		Action: db.Action(action.Value[0]),
		Users:  make([]string, 0, len(users.Values)),
		Paths:  make([]string, 0, len(paths.Values)),
	}

	for _, u := range users.Values {
		username, ok := u.(*wire.String)

		if !ok {
			return wire.NewError("ARG", "Usernames must be strings, got %v", u.Name())
		}

		policy.Users = append(policy.Users, username.Value)
	}

	for _, p := range paths.Values {
		path, ok := p.(*wire.String)

		if !ok {
			return wire.NewError("ARG", "Paths must be strings, got %v", p.Name())
		}

		policy.Paths = append(policy.Paths, "/"+strings.Trim(path.Value, "/"))
	}

	tx := flydb.Txn()
	defer tx.Complete()

	if err := tx.PutAccessPolicy(policy); err != nil {
		log.Fatalf("Failed to create policy: %v", err)
	}

	return wire.OK
}
