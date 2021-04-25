package main

func handlePing(args []string, s *session) {
	s.writer.writeSimpleString("PONG")
}

func handleQuit(args []string, s *session) {
	s.terminated = true
	s.writer.writeOK()
}
