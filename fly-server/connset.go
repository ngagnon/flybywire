package main

import "github.com/ngagnon/fly-server/wire"

func handleConnset(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 2 {
		return wire.NewError("ARG", "Command CONNSET expects exactly 2 arguments")
	}

	if !s.singleUser && s.username == "" {
		return wire.NewError("ILLEGAL", "Cannot change connection settings when unauthenticated")
	}

	key, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Key should be a string, got %s", args[0].Name())
	}

	switch key.Value {
	case "ChunkSize":
		return setChunkSize(args[1], s)
	default:
		return wire.NewError("ARG", "Unknown key")
	}
}

func setChunkSize(val wire.Value, s *sessionInfo) wire.Value {
	size, ok := val.(*wire.Integer)

	if !ok {
		return wire.NewError("ARG", "ChunkSize should be an integer, got %s", val.Name())
	}

	if size.Value <= 0 {
		return wire.NewError("ARG", "ChunkSize must be a positive, non-zero integer")
	}

	if s.session.NumStreams() > 0 {
		return wire.NewError("ILLEGAL", "Cannot set ChunkSize will a stream is open")
	}

	s.session.ChunkSize = size.Value

	return wire.OK
}
