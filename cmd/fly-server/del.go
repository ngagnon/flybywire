package main

import (
	"errors"
	"os"
	"strings"

	log "github.com/ngagnon/flybywire/internal/logging"
	"github.com/ngagnon/flybywire/internal/vfs"
	"github.com/ngagnon/flybywire/internal/wire"
)

func handleDel(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command DEL expects exactly one argument")
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

	if !ok {
		return wire.NewError("NOTFOUND", "No such file or directory")
	}

	info, err := os.Stat(realPath)

	if errors.Is(err, os.ErrNotExist) {
		return wire.NewError("NOTFOUND", "No such file or directory")
	}

	if err != nil {
		log.Debugf("Could not stat a file: %v", err)
		return wire.NewError("ERR", "Unexpected error occurred")
	}

	if info.IsDir() {
		err = os.RemoveAll(realPath)
	} else {
		err = os.Remove(realPath)
	}

	if err != nil {
		log.Debugf("Could not remove file or directory: %v", err)
		return wire.NewError("ERR", "Unexpected error occurred")
	}

	return wire.OK
}
