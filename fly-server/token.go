package main

import (
	"bytes"
	"encoding/ascii85"
	"time"

	"github.com/ngagnon/fly-server/crypto"
	"github.com/ngagnon/fly-server/wire"
)

func handleToken(args []wire.Value, s *sessionInfo) wire.Value {
	if s.singleUser {
		return wire.NewError("ILLEGAL", "Cannot create an authentication token in single-user mode")
	}

	if s.username == "" {
		return wire.NewError("DENIED", "Cannot create an authentication token without being authenticated")
	}

	if tokenKey == nil {
		return wire.NewError("ERROR", "Token authentication is not supported at this time")
	}

	buf := new(bytes.Buffer)
	username := wire.NewString(s.username)
	expiry := wire.NewString(time.Now().Add(5 * time.Minute).UTC().Format(time.RFC3339Nano))
	payload := wire.NewArray([]wire.Value{username, expiry})
	payload.WriteTo(buf)

	plaintext := buf.Bytes()
	encrypted, err := crypto.AesEncrypt(plaintext, tokenKey)

	if err != nil {
		// @TODO: debug log
		return wire.NewError("ERROR", "An unexpected error occurred")
	}

	encoded := make([]byte, ascii85.MaxEncodedLen(len(encrypted)))
	n := ascii85.Encode(encoded, encrypted)
	encoded = encoded[:n]

	return wire.NewString(string(encoded))
}
