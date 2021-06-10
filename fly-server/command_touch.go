package main

import (
	"strings"

	"github.com/ngagnon/fly-server/wire"
)

func handleTouch(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command TOUCH expects exactly one argument")
	}

	rawPath, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Path should be a string, got %s", args[0].Name())
	}

	vPath := "/" + strings.Trim(rawPath.Value, "/")

	if !checkAuth(s, vPath, true) {
		return wire.NewError("DENIED", "Access denied")
	}

	/*
		realPath, ok := resolveVirtualPath(vPath)

		if !ok {
			return wire.NewError("NOTFOUND", "No such file or directory")
		}

			now := time.Now()

			if err := os.Chtimes(realPath, now, now); err != nil {
				// @TODO: debug log
				return wire.NewError("ERR", "Unexpected error occurred")
			}
	*/

	return wire.NewString("OK")
}
