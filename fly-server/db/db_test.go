package db

import (
	"os"
	"path"
	"testing"
	"time"
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

	certPath := path.Join(db.dir, ".fly/cert.pem")

	if _, err := os.Stat(certPath); err != nil {
		t.Fatalf("TLS certificate was not created")
	}

	keyPath := path.Join(db.dir, ".fly/key.pem")

	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("TLS private key was not created")
	}
}

func TestCertAboutToExpire(t *testing.T) {
	dir, err := os.MkdirTemp("", "fly")

	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}

	defer os.RemoveAll(dir)
	db, err := Open(dir)

	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	generateCert(db, 30*time.Minute)

	certPath := path.Join(db.dir, ".fly/cert.pem")
	certInfo, err := os.Stat(certPath)

	if err != nil {
		t.Fatalf("Failed to stat certificate: %v", err)
	}

	keyPath := path.Join(db.dir, ".fly/key.pem")
	keyInfo, err := os.Stat(keyPath)

	if err != nil {
		t.Fatalf("Failed to stat private key: %v", err)
	}

	_, err = Open(dir)

	if err != nil {
		t.Fatalf("Failed to open DB for second time: %v", err)
	}

	newCertInfo, err := os.Stat(certPath)

	if err != nil {
		t.Fatalf("Failed to stat certificate for second time: %v", err)
	}

	if !newCertInfo.ModTime().After(certInfo.ModTime()) {
		t.Fatalf("Should have regenerated certificate")
	}

	newKeyInfo, err := os.Stat(keyPath)

	if err != nil {
		t.Fatalf("Failed to stat private key for second time: %v", err)
	}

	if !newKeyInfo.ModTime().After(keyInfo.ModTime()) {
		t.Fatalf("Should have regenerated private key")
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
