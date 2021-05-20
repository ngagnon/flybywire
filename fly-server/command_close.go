package main

import (
	"github.com/ngagnon/fly-server/session"
	"github.com/ngagnon/fly-server/wire"
)

func handleClose(args []wire.Value, s *session.S) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command CLOSE expects exactly one argument")
	}

	streamId, ok := args[0].(*wire.Integer)

	if !ok {
		return wire.NewError("ARG", "Command CLOSE expects an integer as first argument")
	}

	ok = s.CloseStream(streamId.Value)

	if !ok {
		return wire.NewError("ARG", "Stream is already closed")
	}

	return wire.OK
}
