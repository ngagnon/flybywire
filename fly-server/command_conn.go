package main

func handlePing(args []string, s *session) {
	s.writer.Write([]byte("+PONG\r\n"))
}

func handleQuit(args []string, s *session) {
	s.terminated = true
	s.writer.Write([]byte("+OK\r\n"))
}
