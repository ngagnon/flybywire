package vfs

import (
	"errors"
	"path"
	"strings"

	"github.com/ngagnon/fly-server/db"
)

var ErrInvalid = errors.New("invalid")
var ErrReserved = errors.New("reserved")
var ErrDenied = errors.New("denied")

type PolicyStore interface {
	GetPolicies(path string, username string, action db.Action) []db.Policy
}

var store PolicyStore
var rootDir string

func Setup(policies PolicyStore, rootFolder string) {
	store = policies
	rootDir = rootFolder
}

func ResolveSingleUser(vPath string) (realPath string, err error) {
	return resolve(vPath, nil, nil)
}

func Resolve(vPath string, user *db.User, action db.Action) (realPath string, err error) {
	return resolve(vPath, user, &action)
}

func resolve(vPath string, user *db.User, action *db.Action) (realPath string, err error) {
	cleanPath := "/" + strings.Trim(vPath, "/")
	segments := strings.Split(cleanPath, "/")

	for _, s := range segments {
		st := strings.TrimSpace(s)

		if st == "." || st == ".." {
			return "", ErrInvalid
		}
	}

	if user != nil {
		cleanPath = path.Join(user.Chroot, cleanPath)
		cleanPath = "/" + strings.Trim(cleanPath, "/")
	}

	if action != nil && !authorize(user, cleanPath, *action) {
		return "", ErrDenied
	}

	realPath = path.Join(rootDir, cleanPath)
	flyRoot := path.Join(rootDir, ".fly")

	if strings.HasPrefix(realPath, flyRoot) {
		return "", ErrReserved
	}

	return realPath, nil
}

func authorize(user *db.User, cleanPath string, action db.Action) bool {
	if user == nil {
		return false
	}

	if user.Admin {
		return true
	}

	policies := store.GetPolicies(cleanPath, user.Username, action)

	// Implicit deny
	if len(policies) == 0 {
		return false
	}

	// Explicit deny
	for _, p := range policies {
		if p.Verb == db.Deny {
			return false
		}
	}

	return true
}
