package main

import (
	"errors"
	"os"
	"strings"

	log "github.com/ngagnon/flybywire/internal/logging"
	"github.com/ngagnon/flybywire/internal/vfs"
	"github.com/ngagnon/flybywire/internal/wire"
)

func handleCopy(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 2 {
		return wire.NewError("ARG", "Command COPY expects exactly 2 arguments")
	}

	srcRaw, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Source should be a string, got %s", args[0].Name())
	}

	dstRaw, ok := args[1].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Destination should be a string, got %s", args[1].Name())
	}

	srcRaw.Value = "/" + strings.Trim(srcRaw.Value, "/")
	src, srcErr := resolveRead(s, srcRaw.Value)

	dstRaw.Value = "/" + strings.Trim(dstRaw.Value, "/")
	dst, dstErr := resolveWrite(s, dstRaw.Value)

	if errors.Is(srcErr, vfs.ErrDenied) || errors.Is(dstErr, vfs.ErrDenied) {
		return wire.NewError("DENIED", "Access denied")
	}

	if srcErr != nil || dstErr != nil {
		return wire.NewError("NOTFOUND", "No such file or directory")
	}

	info, err := os.Stat(src)

	if errors.Is(err, os.ErrNotExist) {
		return wire.NewError("NOTFOUND", "No such file or directory")
	}

	if err != nil {
		log.Debugf("Could not stat a file: %v", err)
		return wire.NewError("ERROR", "An unexpected error occurred")
	}

	if !info.Mode().IsRegular() {
		return wire.NewError("ARG", "Source should be a regular file")
	}

	id, wireErr := s.session.NewCopyStream(src, dst)

	if err != nil {
		return wireErr
	}

	return wire.NewInteger(id)
}
