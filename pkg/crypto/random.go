package crypto

import (
	"context"
	"crypto/rand"
	"encoding/base64"
)

func Base64String(size int) (string, error) {
	bytes := make([]byte, size)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

func NewRandomKey(ctx context.Context, encryptor Encryptor, name string) (string, []byte, error) {
	plainKey, _ := Base64String(32)
	encrypted, err := encryptor.Encrypt(ctx, []byte(plainKey), []byte(name))
	if err != nil {
		return "", nil, err
	}

	return plainKey, encrypted, nil
}
