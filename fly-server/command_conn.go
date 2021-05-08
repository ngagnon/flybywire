package main

func handlePing(args []respValue, s *session) respValue {
	return &respString{val: "PONG"}
}

func handleQuit(args []respValue, s *session) respValue {
	s.terminated = true
	return RespOK
}
