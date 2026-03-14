package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
)

func VerifySignature(key []byte, data []byte, signature string) error {
	return verifyHMACSignature(sha256.New, key, data, signature)
}

func VerifySignatureSHA512(key []byte, data []byte, signature string) error {
	return verifyHMACSignature(sha512.New, key, data, signature)
}

func verifyHMACSignature(hashFn func() hash.Hash, key []byte, data []byte, signature string) error {
	h := hmac.New(hashFn, key)
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
