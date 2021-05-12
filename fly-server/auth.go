package main

func checkAuth(s *session, path string, write bool) bool {
	tx := flydb.RTxn()
	defer tx.Complete()

	if tx.NumUsers() == 0 {
		return true
	}

	if s.user == "" {
		return false
	}

	// User doesn't exist anymore
	user, ok := tx.FindUser(s.user)

	if !ok {
		s.user = ""
		return false
	}

	if user.Admin {
		return true
	}

	// @TODO: check ACPs

	return true
}

func checkAdmin(s *session) bool {
	tx := flydb.RTxn()
	defer tx.Complete()

	if tx.NumUsers() == 0 {
		return true
	}

	if s.user == "" {
		return false
	}

	// User doesn't exist anymore
	user, ok := tx.FindUser(s.user)

	if !ok {
		s.user = ""
		return false
	}

	return user.Admin
}
