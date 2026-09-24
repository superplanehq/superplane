// Package opaquetoken generates and hashes high-entropy bearer tokens.
package opaquetoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// Generate returns a 256-bit URL-safe bearer token.
func Generate() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

// Hash returns the stable SHA-256 digest stored by the broker.
func Hash(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}
