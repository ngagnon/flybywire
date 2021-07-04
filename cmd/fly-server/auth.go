package main

import (
	"bytes"
	"encoding/ascii85"
	"time"

	"github.com/ngagnon/flybywire/internal/crypto"
	"github.com/ngagnon/flybywire/internal/wire"
	"golang.org/x/crypto/bcrypt"
)

func handleAuth(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) == 0 {
		return wire.NewError("ARG", "Command AUTH expects at least 1 argument")
	}

	authType, ok := args[0].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "AUTH type should be a string, got %s", args[0].Name())
	}

	switch authType.Value {
	case "PWD":
		return handlePasswordAuth(args, s)
	case "TOK":
		return handleTokenAuth(args, s)
	default:
		return wire.NewError("ARG", "Unsupported AUTH type: %s", authType.Value)
	}
}

func handlePasswordAuth(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 3 {
		return wire.NewError("ARG", "Password authentication requires a username and a password")
	}

	username, ok := args[1].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Username should be a string, got %s", args[1].Name())
	}

	password, ok := args[2].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Password should be a string, got %s", args[2].Name())
	}

	if !verifyPassword(username.Value, password.Value) {
		return wire.NewError("DENIED", "Authentication failed")
	}

	s.changeUser(username.Value)
	return wire.OK
}

func verifyPassword(username string, password string) bool {
	tx := flydb.RTxn()
	user, ok := tx.FindUser(username)
	tx.Complete()

	if !ok {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))

	return err == nil
}

func handleTokenAuth(args []wire.Value, s *sessionInfo) wire.Value {
	if len(args) != 2 {
		return wire.NewError("ARG", "Token authentication requires a token")
	}

	token, ok := args[1].(*wire.String)

	if !ok {
		return wire.NewError("ARG", "Token should be a string, got %s", args[1].Name())
	}

	if tokenKey == nil {
		return wire.NewError("ERROR", "Token authentication is not supported at this time")
	}

	username, ok := verifyToken(token.Value)

	if !ok {
		return wire.NewError("DENIED", "Authentication failed")
	}

	s.changeUser(username)
	return wire.OK
}

func verifyToken(token string) (username string, ok bool) {
	decoded := make([]byte, len(token))
	n, _, err := ascii85.Decode(decoded, []byte(token), true)
	decoded = decoded[:n]

	if err != nil {
		return "", false
	}

	decrypted, err := crypto.AesDecrypt(decoded, tokenKey)

	if err != nil {
		return "", false
	}

	buf := bytes.NewBuffer(decrypted)
	val, err := wire.ReadValue(buf)

	if err != nil {
		return "", false
	}

	payload, ok := val.(*wire.Array)

	if !ok || len(payload.Values) != 2 {
		return "", false
	}

	expiry, ok := payload.Values[1].(*wire.String)

	if !ok {
		return "", false
	}

	date, err := time.Parse(time.RFC3339Nano, expiry.Value)

	if err != nil || time.Now().After(date) {
		return "", false
	}

	usernameString, ok := payload.Values[0].(*wire.String)

	if !ok {
		return "", false
	}

	return usernameString.Value, true
}
