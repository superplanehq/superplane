package encryptor

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
)

type AESGCMEncryptor struct {
	key []byte
}

func NewAESGCMEncryptor(key []byte) Encryptor {
	return &AESGCMEncryptor{key: key}
}

func (e *AESGCMEncryptor) Encrypt(ctx context.Context, data []byte, associatedData []byte) ([]byte, error) {
	aes, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		return nil, err
	}

	// We generate a random nonce for every encryption
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}

	// The final value is nonce+ciphertext
	ciphertext := gcm.Seal(nonce, nonce, data, associatedData)
	return ciphertext, nil
}

func (e *AESGCMEncryptor) Decrypt(ctx context.Context, cyphertext []byte, associatedData []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// We know the nonce is prepended in the cyphertext
	// and we know its size, so can easily separate the two.
	nonceSize := gcm.NonceSize()
	nonce := cyphertext[:nonceSize]
	ciphertext := cyphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, associatedData)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
