package db

import (
	"os"
	"testing"
)

func TestEmptyFolder(t *testing.T) {
	dir, err := os.MkdirTemp("", "fly")

	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}

	defer os.RemoveAll(dir)

	db, err := Open(dir)

	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	tx := db.RTxn()
	defer tx.Complete()

	if tx.NumUsers() != 0 {
		t.Fatal("An empty Fly DB should not have any users")
	}
}

func TestUsers(t *testing.T) {
	dir, err := os.MkdirTemp("", "fly")

	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}

	defer os.RemoveAll(dir)

	db, err := Open(dir)

	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	tx := db.Txn()
	tx.AddUser(&User{
		Username: "john",
		Password: []byte("$2y$12$HsMz8/YX5dIZCM6E99Vw0eeeMRpAUMYHCKkknUhug2vdAEPkNYP6i"),
		Chroot:   "",
		Admin:    true,
	})
	tx.Complete()

	db, err = Open(dir)

	if err != nil {
		t.Fatalf("Failed to open DB for the second time: %v", err)
	}

	rtx := db.RTxn()
	defer rtx.Complete()

	if rtx.NumUsers() != 1 {
		t.Fatal("The updated DB should have one user in it")
	}

	users := rtx.FetchAllUsers()

	if len(users) != 1 {
		t.Fatal("FetchAllUsers should have returned 1 user")
	}

	user := users[0]

	if user.Username != "john" {
		t.Fatal("User john was not found")
	}
}

func TestAccessRules(t *testing.T) {
	dir, err := os.MkdirTemp("", "fly")

	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}

	defer os.RemoveAll(dir)

	db, err := Open(dir)

	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	tx := db.Txn()
	tx.PutAccessPolicy(&Policy{
		Name:   "Path with colon",
		Verb:   Allow,
		Action: Read,
		Users:  []string{"john"},
		Paths:  []string{"/the/first:path", "/the/second/path"},
	})
	tx.Complete()

	db, err = Open(dir)

	if err != nil {
		t.Fatalf("Failed to open DB for the second time: %v", err)
	}

	rtx := db.RTxn()
	defer rtx.Complete()

	policies := rtx.FetchAllPolicies()

	if len(policies) != 1 {
		t.Fatal("FetchAllPolicies should have returned 1 policy")
	}

	policy := policies[0]

	if policy.Name != "Path with colon" {
		t.Fatal("The policy was not found")
	}

	if len(policy.Paths) != 2 {
		t.Fatalf("There should have been 2 paths to the policy, found %d", len(policy.Paths))
	}

	if policy.Paths[0] != "/the/first:path" {
		t.Fatalf("Unexpected path %s", policy.Paths[0])
	}

	if policy.Paths[1] != "/the/second/path" {
		t.Fatalf("Unexpected path %s", policy.Paths[1])
	}
}
