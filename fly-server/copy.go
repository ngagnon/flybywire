package main

import (
	"errors"
	"os"
	"strings"

	"github.com/ngagnon/fly-server/wire"
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
	src, srcOk := resolveVirtualPath(srcRaw.Value, s.user)

	dstRaw.Value = "/" + strings.Trim(dstRaw.Value, "/")
	dst, dstOk := resolveVirtualPath(dstRaw.Value, s.user)

	if !srcOk || !dstOk {
		return wire.NewError("NOTFOUND", "No such file or directory")
	}

	if !checkAuth(s, srcRaw.Value, true) || !checkAuth(s, dstRaw.Value, true) {
		return wire.NewError("DENIED", "Access denied")
	}

	info, err := os.Stat(src)

	if errors.Is(err, os.ErrNotExist) {
		return wire.NewError("NOTFOUND", "No such file or directory")
	}

	if err != nil {
		// @TODO: debug log?
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
