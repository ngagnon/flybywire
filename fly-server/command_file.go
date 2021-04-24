package main

import (
	"os"
	"path"
	"strings"
)

func handleMkdir(args []string, s *session) {
	if len(args) != 1 {
		msg := "-ERR Command MKDIR expects exactly one argument\r\n"
		s.writer.Write([]byte(msg))
		return
	}

	vPath := "/" + strings.Trim(args[0], "/")

	if !checkAuth(s, vPath, true) {
		s.writer.Write([]byte("-DENIED\r\n"))
		return
	}

	realPath := path.Join(dir, strings.TrimPrefix(vPath, "/"))

	if err := os.MkdirAll(realPath, 0755); err != nil {
		panic(err)
	}

	s.writer.Write([]byte("+OK\r\n"))
}
