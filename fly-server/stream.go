package main

import (
	"strings"

	"github.com/ngagnon/fly-server/wire"
)

func handleStream(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 2 {
		return wire.NewError("ARG", "Command STREAM expects exactly 2 arguments")
	}

	mode, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Mode should be a string, got %s", args[0].Name())
	}

	rawPath, ok := args[1].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Path should be a string, got %s", args[1].Name())
	}

	vPath := "/" + strings.TrimPrefix(rawPath.Value, "/")

	if mode.Value != "W" && mode.Value != "R" {
		return wire.NewError("ARG", "Unsupported mode: %s", mode.Value)
	}

	writing := mode.Value == "W"

	if !checkAuth(s, vPath, writing) {
		return wire.NewError("DENIED", "Access denied")
	}

	realPath, ok := resolveVirtualPath(vPath, s.user)

	if !ok {
		return wire.NewError("NOTFOUND", "No such file or directory")
	}

	if writing {
		id, err := s.session.NewWriteStream(realPath)

		if err != nil {
			return err
		}

		return wire.NewInteger(id)
	} else {
		id, err := s.session.NewReadStream(realPath)

		if err != nil {
			return err
		}

		return wire.NewInteger(id)
	}
}
