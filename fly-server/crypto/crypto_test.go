package crypto

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
)

func TestAes(t *testing.T) {
	key, err := RandomKey(16)

	if err != nil {
		t.Fatalf("Error occurred while generating random key: %v", err)
	}

	msg := gofakeit.LoremIpsumParagraph(10, 10, 40, "\n")

	encrypted, err := AesEncrypt([]byte(msg), key)

	if err != nil {
		t.Fatalf("Error occurred while encrypting data: %v", err)
	}

	decrypted, err := AesDecrypt(encrypted, key)

	if err != nil {
		t.Fatalf("Error occurred while decrypting data: %v", err)
	}

	if string(decrypted) != msg {
		t.Fatal("The original and decrypted messages don't match")
	}
}
