package main

import (
	"errors"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ngagnon/fly-server/wire"
)

func handleList(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command LIST expects exactly one argument")
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

	info, err := os.Stat(realPath)

	if errors.Is(err, os.ErrNotExist) {
		return wire.NewError("NOTFOUND", "No such file or directory")
	}

	if err != nil {
		// @TODO: debug log
		return wire.NewError("ERR", "Unexpected error occurred")
	}

	table := &wire.Table{}

	if info.IsDir() {
		files, err := os.ReadDir(realPath)

		if err != nil {
			// @TODO: debug log
			return wire.NewError("ERR", "Unexpected error occurred")
		}

		for _, file := range files {
			info, err := file.Info()

			if err != nil {
				// @TODO: debug log
				return wire.NewError("ERR", "Unexpected error occurred")
			}

			fullPath := path.Join(realPath, info.Name())
			addFile(table, info, fullPath)
		}
	} else {
		addFile(table, info, realPath)
	}

	return table
}

func addFile(t *wire.Table, info os.FileInfo, fullPath string) {
	var ftype string
	var fsize wire.Value

	if isReservedPath(fullPath) {
		return
	}

	if info.IsDir() {
		ftype = "D"
		fsize = wire.Null
	} else if info.Mode().IsRegular() {
		ftype = "F"
		fsize = wire.NewInteger(int(info.Size()))
	} else {
		return
	}

	t.Add([]wire.Value{
		wire.NewString(ftype),
		wire.NewString(info.Name()),
		fsize,
		wire.NewString(info.ModTime().UTC().Format(time.RFC3339Nano)),
	})
}
