package main

import (
	"path"
	"strings"
)

func resolveVirtualPath(vPath string) (realPath string, ok bool) {
	cleanPath := strings.Trim(vPath, "/")

	if cleanPath == ".fly" || strings.HasPrefix(cleanPath, ".fly/") {
		return "", false
	}

	return path.Join(dir, cleanPath), true
}
