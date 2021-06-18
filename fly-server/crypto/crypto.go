package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func RandomKey(size int) ([]byte, error) {
	key := make([]byte, size)
	_, err := rand.Read(key)
	return key, err
}

func AesEncrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)

	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	output := gcm.Seal(nonce, nonce, plaintext, nil)

	return output, nil
}

func AesDecrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)

	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)

	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
