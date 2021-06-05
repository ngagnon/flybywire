package main

func checkAuth(s *sessionInfo, path string, write bool) bool {
	if s.singleUser {
		return true
	}

	if s.username == "" {
		return false
	}

	if s.user.Admin {
		return true
	}

	// @TODO: check ACPs

	return true
}

func checkAdmin(s *sessionInfo) bool {
	if s.singleUser {
		return true
	}

	if s.username == "" {
		return false
	}

	return s.user.Admin
}
