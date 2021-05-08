package main

import "github.com/ngagnon/fly-server/wire"

func handlePing(args []wire.Value, s *session) wire.Value {
	return wire.NewString("PONG")
}

func handleQuit(args []wire.Value, s *session) wire.Value {
	s.terminated = true
	return wire.OK
}
