package main

import (
	"path"
	"strings"

	"github.com/ngagnon/fly-server/db"
)

func resolveVirtualPath(vPath string, user *db.User) (realPath string, ok bool) {
	cleanPath := strings.Trim(vPath, "/")
	segments := strings.Split(cleanPath, "/")

	for _, s := range segments {
		st := strings.TrimSpace(s)

		if st == "." || st == ".." {
			return "", false
		}
	}

	realPath = dir

	if user != nil {
		realPath = path.Join(realPath, user.Chroot)
	}

	realPath = path.Join(realPath, cleanPath)
	flyRoot := path.Join(dir, ".fly")

	if strings.HasPrefix(realPath, flyRoot) {
		return "", false
	}

	return realPath, true
}

func isReservedPath(realPath string) bool {
	flyRoot := path.Join(dir, ".fly")
	return strings.HasPrefix(realPath, flyRoot)
}
