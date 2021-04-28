package main

import (
	"os"
	"path"
	"strings"
)

func handleMkdir(args []string, s *session) error {
	if len(args) != 1 {
		return s.writeError("ERR", "Command MKDIR expects exactly one argument")
	}

	vPath := "/" + strings.Trim(args[0], "/")

	if !checkAuth(s, vPath, true) {
		return s.writeError("DENIED", "Access denied")
	}

	realPath := path.Join(dir, strings.TrimPrefix(vPath, "/"))

	if err := os.MkdirAll(realPath, 0755); err != nil {
		return err
	}

	return s.writeOK()
}
