package main

func handlePing(args []string, s *session) error {
	return s.writeSimpleString("PONG")
}

func handleQuit(args []string, s *session) error {
	s.terminated = true
	return s.writeOK()
}
