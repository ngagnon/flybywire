package main

import (
	"path"
	"strings"
)

func resolveVirtualPath(vPath string) string {
	cleanPath := strings.Trim(vPath, "/")
	return path.Join(dir, cleanPath)
}

func isReservedPath(realPath string) bool {
	flyRoot := path.Join(dir, ".fly")
	return strings.HasPrefix(realPath, flyRoot)
}
