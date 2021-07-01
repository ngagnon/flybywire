package main

import (
	"github.com/ngagnon/fly-server/db"
	"github.com/ngagnon/fly-server/vfs"
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

// @TODO: maybe not needed?
func checkAdmin(s *sessionInfo) bool {
	if s.singleUser {
		return true
	}

	if s.username == "" {
		return false
	}

	return s.user.Admin
}
