package main

import "github.com/ngagnon/fly-server/wire"

func handleConnget(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command CONNGET expects exactly one argument")
	}

	if !s.singleUser && s.username == "" {
		return wire.NewError("ILLEGAL", "Cannot read connection settings when unauthenticated")
	}

	key, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Key should be a string, got %s", args[0].Name())
	}

	switch key.Value {
	case "ChunkSize":
		return wire.NewInteger(s.session.ChunkSize)
	default:
		return wire.NewError("ARG", "Unknown key")
	}
}
