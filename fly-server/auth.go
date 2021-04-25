package main

func checkAuth(s *session, path string, write bool) bool {
	globalLock.RLock()
	defer globalLock.RUnlock()

	if singleUser {
		return true
	}

	if s.user == "" {
		return false
	}

	// User doesn't exist anymore
	if _, ok := users[s.user]; !ok {
		s.user = ""
		return false
	}

	if users[s.user].admin {
		return true
	}

	// @TODO: check ACPs

	return true
}

func checkAdmin(s *session) bool {
	globalLock.RLock()
	defer globalLock.RUnlock()

	if singleUser {
		return true
	}

	if s.user == "" {
		return false
	}

	// User doesn't exist anymore
	if _, ok := users[s.user]; !ok {
		s.user = ""
		return false
	}

	return users[s.user].admin
}
