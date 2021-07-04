package main

import (
	"errors"
	"os"
	"strings"

	log "github.com/ngagnon/flybywire/internal/logging"
	"github.com/ngagnon/flybywire/internal/vfs"
	"github.com/ngagnon/flybywire/internal/wire"
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
	realPath, err := resolveWrite(s, vPath)

	if errors.Is(err, vfs.ErrDenied) {
		return wire.NewError("DENIED", "Access denied")
	}

	if errors.Is(err, vfs.ErrInvalid) || errors.Is(err, vfs.ErrReserved) {
		return wire.NewError("NOTFOUND", "No such file or directory")
	}

	if err := os.MkdirAll(realPath, 0755); err != nil {
		log.Debugf("Could not create folder: %v", err)
		return wire.NewError("ERR", "Unexpected error occurred")
	}

	return wire.OK
}
