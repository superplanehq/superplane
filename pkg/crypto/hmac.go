package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
)

func VerifySignature(key []byte, data []byte, signature string) error {
	h := hmac.New(sha256.New, key)
	h.Write(data)

	computed := fmt.Sprintf("%x", h.Sum(nil))
	if !hmac.Equal([]byte(computed), []byte(signature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

func Sign(key []byte, data []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func VerifySignatureSHA512(key []byte, data []byte, signature string) error {
	h := hmac.New(sha512.New, key)
	h.Write(data)

	computed := fmt.Sprintf("%x", h.Sum(nil))
	if computed != signature {
		return fmt.Errorf("invalid signature")
	}

	return nil
}
