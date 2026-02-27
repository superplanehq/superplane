package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
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
	expected := h.Sum(nil)

	// Decode the provided signature from hex for constant-time comparison.
	sigBytes, err := hex.DecodeString(signature)
	if err != nil || !hmac.Equal(expected, sigBytes) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}
