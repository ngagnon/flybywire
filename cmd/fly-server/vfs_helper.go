package main

import (
	"github.com/ngagnon/flybywire/internal/db"
	"github.com/ngagnon/flybywire/internal/vfs"
)

func resolveRead(s *sessionInfo, path string) (realPath string, err error) {
	return resolve(s, path, false)
}

func resolveWrite(s *sessionInfo, path string) (realPath string, err error) {
	return resolve(s, path, true)
}

func resolve(s *sessionInfo, path string, write bool) (realPath string, err error) {
	if s.singleUser {
		return vfs.ResolveSingleUser(path)
	}

	action := db.Read

	if write {
		action = db.Write
	}

	return vfs.Resolve(path, s.user, action)
}
