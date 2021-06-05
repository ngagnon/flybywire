package main

import (
	"errors"
	"regexp"

	"github.com/ngagnon/fly-server/db"
	log "github.com/ngagnon/fly-server/logging"
	"github.com/ngagnon/fly-server/wire"
	"golang.org/x/crypto/bcrypt"
)

func handleListUser(args []wire.Value, s *sessionInfo) wire.Value {
	if !checkAdmin(s) {
		return wire.NewError("DENIED", "You are not allowed to manage users")
	}

	tx := flydb.RTxn()
	users := tx.FetchAllUsers()
	tx.Complete()

	usernames := make([]wire.Value, 0, len(users))

	for _, u := range users {
		usernames = append(usernames, wire.NewString(u.Username))
	}

	return wire.NewArray(usernames)
}

func handleAddUser(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 2 {
		return wire.NewError("ARG", "Command ADDUSER expects exactly 2 arguments")
	}

	if !checkAdmin(s) {
		return wire.NewError("DENIED", "You are not allowed to manage users")
	}

	username, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Username should be a string, got %s", args[0].Name())
	}

	password, ok := args[1].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Password should be a string, got %s", args[1].Name())
	}

	if len(username.Value) < 1 {
		return wire.NewError("ARG", "Minimum username length is 1")
	}

	if len(username.Value) > 32 {
		return wire.NewError("ARG", "Maximum username length is 32")
	}

	if matched, err := regexp.Match("^[a-z_]([a-z0-9_-]{0,31})$", []byte(username.Value)); !matched || err != nil {
		return wire.NewError("ARG", "Invalid username")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password.Value), 12)

	if err != nil {
		log.Errorf("Unexpected error while generating hash: %v", err)
		return wire.NewError("ERR", "Unexpected error while generating hash")
	}

	newUser := &db.User{
		Username: username.Value,
		Password: hash,
		Chroot:   "",
		Admin:    s.singleUser,
	}

	tx := flydb.Txn()

	if err = tx.AddUser(newUser); err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	tx.Complete()

	if s.singleUser {
		s.changeUser(newUser.Username)
	}

	return wire.OK
}

func handleWhoAmI(args []wire.Value, s *sessionInfo) wire.Value {
	username := s.username

	if username == "" {
		return wire.Null
	}

	return wire.NewString(username)
}

func handleShowUser(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command SHOWUSER expects exactly 1 argument")
	}

	if !checkAdmin(s) {
		return wire.NewError("DENIED", "You are not allowed to manage users.")
	}

	username, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Username should be a string, got %s", args[0].Name())
	}

	tx := flydb.RTxn()
	user, ok := tx.FindUser(username.Value)
	tx.Complete()

	if !ok {
		return wire.NewError("NOTFOUND", "User not found")
	}

	result := make(map[string]wire.Value)
	result["username"] = wire.NewString(user.Username)
	result["chroot"] = wire.NewString(user.Chroot)
	result["admin"] = wire.NewBoolean(user.Admin)

	return wire.NewMap(result)
}

func handleRmuser(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 1 {
		return wire.NewError("ARG", "Command RMUSER expects exactly 1 argument")
	}

	if !checkAdmin(s) {
		return wire.NewError("DENIED", "You are not allowed to manage users.")
	}

	username, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Username should be a string, got %s", args[0].Name())
	}

	tx := flydb.Txn()
	err := tx.DeleteUser(username.Value)
	tx.Complete()

	if errors.Is(err, db.ErrNotFound) {
		return wire.NewError("NOTFOUND", "User not found")
	}

	if err != nil {
		log.Fatalf("Failed to delete user user: %v", err)
	}

	return wire.OK
}

func handleSetpwd(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 2 {
		return wire.NewError("ARG", "Command SETPWD expects exactly 2 arguments")
	}

	username, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Username should be a string, got %s", args[0].Name())
	}

	password, ok := args[1].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Password should be a string, got %s", args[1].Name())
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password.Value), 12)

	if err != nil {
		log.Errorf("Unexpected error while generating hash: %v", err)
		return wire.NewError("ERR", "Unexpected error while generating hash")
	}

	if s.singleUser {
		return wire.NewError("ILLEGAL", "Cannot manage users in single-user mode")
	}

	if s.username != username.Value && !checkAdmin(s) {
		return wire.NewError("DENIED", "You are not allowed to manage users.")
	}

	tx := flydb.Txn()
	defer tx.Complete()

	err = tx.UpdateUser(username.Value, func(u *db.User) {
		u.Password = hash
	})

	if err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}

	return wire.OK
}
