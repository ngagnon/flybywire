package main

import "github.com/ngagnon/fly-server/wire"

func handleWhoAmI(args []wire.Value, s *sessionInfo) wire.Value {
	username := s.username

	if username == "" {
		return wire.Null
	}

	return wire.NewString(username)
}
