package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

func VerifySignature(key []byte, data []byte, signature string) error {
	h := hmac.New(sha256.New, key)
	h.Write(data)

	computed := fmt.Sprintf("%x", h.Sum(nil))
	if computed != signature {
		return fmt.Errorf("invalid signature")
	}

	return nil
}
