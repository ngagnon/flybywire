package main

import "github.com/ngagnon/fly-server/session"

func checkAuth(s *session.S, path string, write bool) bool {
	tx := flydb.RTxn()
	defer tx.Complete()

	if tx.NumUsers() == 0 {
		return true
	}

	username := s.CurrentUser()

	if username == "" {
		return false
	}

	// User doesn't exist anymore
	user, ok := tx.FindUser(username)

	if !ok {
		s.SetUser("")
		return false
	}

	if user.Admin {
		return true
	}

	// @TODO: check ACPs

	return true
}

func checkAdmin(s *session.S) bool {
	tx := flydb.RTxn()
	defer tx.Complete()

	if tx.NumUsers() == 0 {
		return true
	}

	username := s.CurrentUser()

	if username == "" {
		return false
	}

	// User doesn't exist anymore
	user, ok := tx.FindUser(username)

	if !ok {
		s.SetUser("")
		return false
	}

	return user.Admin
}
