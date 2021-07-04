package vfs

import (
	"errors"
	"os"
	"testing"

	"github.com/ngagnon/flybywire/internal/db"
)

type policyStore struct {
	policies []db.Policy
}

func (s *policyStore) GetPolicies(path string, username string, action db.Action) []db.Policy {
	return s.policies
}

func setup(store PolicyStore, t *testing.T) {
	dir, err := os.MkdirTemp("", "fly.vfs")

	if err != nil {
		t.Fatalf("Failed to make temp directory: %v", err)
	}

	Setup(store, dir)
}

func TestInvalidPath(t *testing.T) {
	store := &policyStore{policies: make([]db.Policy, 0)}
	setup(store, t)

	_, err := Resolve("../../some/path", nil, db.Read)

	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Resolve should have returned ErrInvalid, got %v", err)
	}
}

func TestResolveUnauthenticated(t *testing.T) {
	store := &policyStore{
		policies: []db.Policy{
			{
				Verb:   db.Allow,
				Action: db.Read,
				Users:  []string{"fooz", "johnnyboy"},
				Paths:  []string{"/home/johnnyboy"},
			},
		},
	}

	setup(store, t)

	_, err := Resolve("/home/johnnyboy/recipes", nil, db.Read)

	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Resolve should have returned ErrDenied, got %v", err)
	}
}

func TestResolveExplicitAllowRead(t *testing.T) {
	store := &policyStore{
		policies: []db.Policy{
			{
				Verb:   db.Allow,
				Action: db.Read,
				Users:  []string{"fooz", "johnnyboy"},
				Paths:  []string{"/home/johnnyboy"},
			},
		},
	}

	setup(store, t)

	user := &db.User{Username: "johnnyboy"}
	_, err := Resolve("/home/johnnyboy/recipes", user, db.Read)

	if err != nil {
		t.Fatalf("Resolve should have allowed the operation, got %v", err)
	}
}

func TestResolveExplicitAllowWrite(t *testing.T) {
	store := &policyStore{
		policies: []db.Policy{
			{
				Verb:   db.Allow,
				Action: db.Write,
				Users:  []string{"fooz", "johnnyboy"},
				Paths:  []string{"/home/johnnyboy"},
			},
		},
	}

	setup(store, t)

	user := &db.User{Username: "johnnyboy"}
	_, err := Resolve("/home/johnnyboy/recipes/jambalaya.txt", user, db.Write)

	if err != nil {
		t.Fatalf("Resolve should have allowed the operation, got %v", err)
	}
}

func TestResolveImplicitDenyRead(t *testing.T) {
	store := &policyStore{policies: make([]db.Policy, 0)}
	setup(store, t)

	user := &db.User{Username: "johnnyboy"}
	_, err := Resolve("/home/johnnyboy/recipes", user, db.Read)

	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Resolve should have returned ErrDenied, got %v", err)
	}
}

func TestResolveImplicitDenyWrite(t *testing.T) {
	store := &policyStore{policies: make([]db.Policy, 0)}
	setup(store, t)

	user := &db.User{Username: "johnnyboy"}
	_, err := Resolve("/home/johnnyboy/recipes/jambalaya.txt", user, db.Write)

	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Resolve should have returned ErrDenied, got %v", err)
	}
}

func TestResolveExplicitDenyRead(t *testing.T) {
	store := &policyStore{
		policies: []db.Policy{
			{
				Verb:   db.Allow,
				Action: db.Read,
				Users:  []string{"fooz", "johnnyboy"},
				Paths:  []string{"/home/johnnyboy"},
			},
			{
				Verb:   db.Deny,
				Action: db.Read,
				Users:  []string{"fooz", "johnnyboy"},
				Paths:  []string{"/home/johnnyboy/recipes"},
			},
			{
				Verb:   db.Allow,
				Action: db.Read,
				Users:  []string{"fooz", "johnnyboy"},
				Paths:  []string{"/home/johnnyboy/recipes/cajun"},
			},
		},
	}

	setup(store, t)

	user := &db.User{Username: "johnnyboy"}
	_, err := Resolve("/home/johnnyboy/recipes/cajun/jambalaya.txt", user, db.Read)

	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Resolve should have returned ErrDenied, got %v", err)
	}
}

func TestResolveExplicitDenyWrite(t *testing.T) {
	store := &policyStore{
		policies: []db.Policy{
			{
				Verb:   db.Allow,
				Action: db.Write,
				Users:  []string{"fooz", "johnnyboy"},
				Paths:  []string{"/home/johnnyboy"},
			},
			{
				Verb:   db.Deny,
				Action: db.Write,
				Users:  []string{"fooz", "johnnyboy"},
				Paths:  []string{"/home/johnnyboy/recipes"},
			},
			{
				Verb:   db.Allow,
				Action: db.Write,
				Users:  []string{"fooz", "johnnyboy"},
				Paths:  []string{"/home/johnnyboy/recipes/cajun"},
			},
		},
	}

	setup(store, t)

	user := &db.User{Username: "johnnyboy"}
	_, err := Resolve("/home/johnnyboy/recipes/cajun/jambalaya.txt", user, db.Write)

	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Resolve should have returned ErrDenied, got %v", err)
	}
}

func TestResolveSingleUser(t *testing.T) {
	store := &policyStore{policies: make([]db.Policy, 0)}
	setup(store, t)

	_, err := ResolveSingleUser("/home/johnnyboy/recipes")

	if err != nil {
		t.Fatalf("ResolveSingleUser should have allowed the operation, got %v", err)
	}
}

func TestAdminUserImplicitDeny(t *testing.T) {
	store := &policyStore{policies: make([]db.Policy, 0)}
	setup(store, t)

	user := &db.User{Username: "johnnyboy", Admin: true}
	_, err := Resolve("/home/johnnyboy/recipes/jambalaya.txt", user, db.Write)

	if err != nil {
		t.Fatalf("Resolve should have allowed the operation, got %v", err)
	}
}

func TestAdminUserExplicitDeny(t *testing.T) {
	store := &policyStore{
		policies: []db.Policy{
			{
				Verb:   db.Allow,
				Action: db.Write,
				Users:  []string{"fooz", "johnnyboy"},
				Paths:  []string{"/home/johnnyboy"},
			},
			{
				Verb:   db.Deny,
				Action: db.Write,
				Users:  []string{"fooz", "johnnyboy"},
				Paths:  []string{"/home/johnnyboy/recipes"},
			},
			{
				Verb:   db.Allow,
				Action: db.Write,
				Users:  []string{"fooz", "johnnyboy"},
				Paths:  []string{"/home/johnnyboy/recipes/cajun"},
			},
		},
	}

	setup(store, t)

	user := &db.User{Username: "johnnyboy", Admin: true}
	_, err := Resolve("/home/johnnyboy/recipes/cajun/jambalaya.txt", user, db.Write)

	if err != nil {
		t.Fatalf("Resolve should have allowed the operation, got %v", err)
	}
}
