package main

import (
	"errors"
	"os"
	"strings"
	"time"

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

	realPath, ok := resolveVirtualPath(vPath, s.user)

	if !ok {
		return wire.NewError("NOTFOUND", "No such file or directory")
	}

	now := time.Now()
	err := os.Chtimes(realPath, now, now)

	if errors.Is(err, os.ErrNotExist) {
		var f *os.File
		f, err = os.Create(realPath)

		if errors.Is(err, os.ErrNotExist) {
			return wire.NewError("NOTFOUND", "No such file or directory")
		}

		if err != nil {
			// @TODO: debug log
			return wire.NewError("ERR", "Unexpected error occurred")
		}

		f.Close()
	}

	if err != nil {
		// @TODO: debug log
		return wire.NewError("ERR", "Unexpected error occurred")
	}

	return wire.OK
}
