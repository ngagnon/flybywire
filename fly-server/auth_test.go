package main

import "testing"

func TestCheckAuth(t *testing.T) {
	users = make(map[string]user)
	sess := &session{
		terminated: false,
		user:       "",
	}

	singleUser = true

	if ok := checkAuth(sess, "/some/path", true); !ok {
		t.Fatal("Should always return true in single-user mode")
	}

	singleUser = false

	if ok := checkAuth(sess, "/some/path", true); ok {
		t.Fatal("Should return false for unauthenticated users")
	}

	sess.user = "bob"

	if ok := checkAuth(sess, "/some/path", true); ok {
		t.Fatal("Should return false for non-existing users")
	}

	if sess.user != "" {
		t.Fatal("Should set user to empty string if did not exist")
	}

	sess.user = "bob"
	users["bob"] = user{admin: true}

	if ok := checkAuth(sess, "/some/path", true); !ok {
		t.Fatal("Should always return true for admin users")
	}
}
