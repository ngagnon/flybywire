package main

import "github.com/ngagnon/fly-server/wire"

func handleListAcp(args []wire.Value, s *sessionInfo) wire.Value {
	if s.singleUser {
		return wire.NewError("ILLEGAL", "Cannot manage ACPs in single-user mode")
	}

	if !checkAdmin(s) {
		return wire.NewError("DENIED", "You are not allowed to manage users")
	}

	tx := flydb.RTxn()
	policies := tx.FetchAllPolicies()
	tx.Complete()

	table := &wire.Table{}

	for _, p := range policies {
		users := make([]wire.Value, 0, len(p.Users))

		for _, u := range p.Users {
			users = append(users, wire.NewString(u))
		}

		paths := make([]wire.Value, 0, len(p.Paths))

		for _, p := range p.Paths {
			paths = append(paths, wire.NewString(p))
		}

		table.Add([]wire.Value{
			wire.NewString(p.Name),
			wire.NewString(string(p.Verb)),
			wire.NewString(string(p.Action)),
			wire.NewArray(users),
			wire.NewArray(paths),
		})
	}

	return table
}
