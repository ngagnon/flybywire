package main

import (
	"github.com/ngagnon/fly-server/wire"
)

func handlePing(args []wire.Value, s *sessionInfo) wire.Value {
	return wire.NewString("PONG")
}
