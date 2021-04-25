package main

import (
	"os"
	"path"
	"strings"
)

func handleMkdir(args []string, s *session) {
	if len(args) != 1 {
		s.writer.writeError("ERR", "Command MKDIR expects exactly one argument")
		return
	}

	vPath := "/" + strings.Trim(args[0], "/")

	if !checkAuth(s, vPath, true) {
		s.writer.writeError("DENIED", "Access denied")
		return
	}

	realPath := path.Join(dir, strings.TrimPrefix(vPath, "/"))

	if err := os.MkdirAll(realPath, 0755); err != nil {
		panic(err)
	}

	s.writer.writeOK()
}
