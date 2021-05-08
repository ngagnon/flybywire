package main

import (
	"path"
	"strings"
)

func resolveVirtualPath(vPath string) string {
	return path.Join(dir, strings.TrimPrefix(vPath, "/"))
}
