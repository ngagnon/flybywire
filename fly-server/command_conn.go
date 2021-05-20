package main

import (
	"github.com/ngagnon/fly-server/session"
	"github.com/ngagnon/fly-server/wire"
)

func handlePing(args []wire.Value, s *session.S) wire.Value {
	return wire.NewString("PONG")
}
