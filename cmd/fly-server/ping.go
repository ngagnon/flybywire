package main

import (
	"github.com/ngagnon/flybywire/internal/wire"
)

func handlePing(args []wire.Value, s *sessionInfo) wire.Value {
	return wire.NewString("PONG")
}
