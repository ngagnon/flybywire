package main

import (
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
	src := resolveVirtualPath(srcRaw.Value)

	dstRaw.Value = "/" + strings.Trim(dstRaw.Value, "/")
	dst := resolveVirtualPath(dstRaw.Value)

	if !checkAuth(s, srcRaw.Value, true) || !checkAuth(s, dstRaw.Value, true) {
		return wire.NewError("DENIED", "Access denied")
	}

	if isReservedPath(src) || isReservedPath(dst) {
		return wire.NewError("NOTFOUND", "No such file or directory")
	}

	id, err := s.session.NewCopyStream(src, dst)

	if err != nil {
		return err
	}

	return wire.NewInteger(id)
}
