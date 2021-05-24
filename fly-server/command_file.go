package main

import (
	"os"
	"strings"

	"github.com/ngagnon/fly-server/session"
	"github.com/ngagnon/fly-server/wire"
)

func handleMkdir(args []wire.Value, s *session.S) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command MKDIR expects exactly one argument")
	}

	pathBlob, ok := args[0].(*wire.Blob)

	if !ok {
		return wire.NewError("ARG", "Path should be a blob, got %s", args[0].Name())
	}

	vPath := "/" + strings.Trim(string(pathBlob.Data), "/")

	if !checkAuth(s, vPath, true) {
		return wire.NewError("DENIED", "Access denied")
	}

	realPath := resolveVirtualPath(vPath)

	if err := os.MkdirAll(realPath, 0755); err != nil {
		// @TODO: debug log
		return wire.NewError("ERR", "Unexpected error occurred")
	}

	return wire.OK
}

func handleStream(args []wire.Value, s *session.S) wire.Value {
	if len(args) != 2 {
		return wire.NewError("ARG", "Command STREAM expects exactly 2 arguments")
	}

	mode, ok := args[0].(*wire.Blob)

	if !ok {
		return wire.NewError("ARG", "Mode should be a blob, got %s", args[0].Name())
	}

	pathBlob, ok := args[1].(*wire.Blob)

	if !ok {
		return wire.NewError("ARG", "Path should be a blob, got %s", args[1].Name())
	}

	vPath := "/" + strings.TrimPrefix(string(pathBlob.Data), "/")

	if string(mode.Data) != "W" && string(mode.Data) != "R" {
		return wire.NewError("ARG", "Unsupported mode: %s", mode.Data)
	}

	writing := string(mode.Data) == "W"

	if !checkAuth(s, vPath, writing) {
		return wire.NewError("DENIED", "Access denied")
	}

	realPath := resolveVirtualPath(vPath)

	if writing {
		id, err := s.NewWriteStream(realPath)

		if err != nil {
			return err
		}

		return wire.NewInteger(id)
	} else {
		id, err := s.NewReadStream(realPath)

		if err != nil {
			return err
		}

		return wire.NewInteger(id)
	}
}
