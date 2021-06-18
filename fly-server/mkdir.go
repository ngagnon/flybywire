package main

import (
	"os"
	"strings"

	"github.com/ngagnon/fly-server/wire"
)

func handleMkdir(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command MKDIR expects exactly one argument")
	}

	rawPath, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Path should be a string, got %s", args[0].Name())
	}

	vPath := "/" + strings.Trim(rawPath.Value, "/")

	if !checkAuth(s, vPath, true) {
		return wire.NewError("DENIED", "Access denied")
	}

	realPath := resolveVirtualPath(vPath)

	if isReservedPath(realPath) {
		return wire.NewError("NOTFOUND", "No such file or directory")
	}

	if err := os.MkdirAll(realPath, 0755); err != nil {
		// @TODO: debug log
		return wire.NewError("ERR", "Unexpected error occurred")
	}

	return wire.OK
}
