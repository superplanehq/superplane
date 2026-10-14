package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"strings"
)

// MasterKeys protects data encryption keys. A reference identifies the
// configured master key; it never contains secret key material.
type MasterKeys interface {
	Wrap(context.Context, string, []byte, []byte) ([]byte, error)
	Unwrap(context.Context, string, []byte, []byte) ([]byte, error)
	Has(string) bool
}

type LocalMasterKeys struct {
	keys map[string]cipher.AEAD
}

func ParseLocalMasterKeys(configured string) (*LocalMasterKeys, error) {
	if configured == "" {
		return nil, fmt.Errorf("ENCRYPTION_LOCAL_MASTER_KEYS is required")
	}

	keys := make(map[string]cipher.AEAD)
	for entry := range strings.SplitSeq(configured, ",") {
		name, value, ok := strings.Cut(entry, ":")
		if !ok || !validMasterKeyID(name) {
			return nil, fmt.Errorf("invalid local master key ID")
		}
		if _, exists := keys[name]; exists {
			return nil, fmt.Errorf("duplicate local master key ID %q", name)
		}
		material, err := base64.StdEncoding.Strict().DecodeString(value)
		if err != nil || len(material) != 32 {
			return nil, fmt.Errorf("local master key %q must contain 32 base64-encoded bytes", name)
		}
		block, err := aes.NewCipher(material)
		if err != nil {
			return nil, fmt.Errorf("create local master key %q: %w", name, err)
		}
		aead, err := cipher.NewGCMWithRandomNonce(block)
		if err != nil {
			return nil, fmt.Errorf("create local master AEAD %q: %w", name, err)
		}
		keys[name] = aead
	}
	return &LocalMasterKeys{keys: keys}, nil
}

func validMasterKeyID(name string) bool {
	if name == "" {
		return false
	}
	for _, character := range name {
		if character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			character == '-' || character == '_' {
			continue
		}
		return false
	}
	return true
}

func (k *LocalMasterKeys) Has(reference string) bool {
	_, ok := k.keys[reference]
	return ok
}

func (k *LocalMasterKeys) Wrap(_ context.Context, reference string, plain, associatedData []byte) ([]byte, error) {
	aead, ok := k.keys[reference]
	if !ok {
		return nil, fmt.Errorf("local master key %q is not configured", reference)
	}
	return aead.Seal(nil, nil, plain, associatedData), nil
}

func (k *LocalMasterKeys) Unwrap(_ context.Context, reference string, wrapped, associatedData []byte) ([]byte, error) {
	aead, ok := k.keys[reference]
	if !ok {
		return nil, fmt.Errorf("local master key %q is not configured", reference)
	}
	plain, err := aead.Open(nil, nil, wrapped, associatedData)
	if err != nil {
		return nil, fmt.Errorf("unwrap with local master key %q: %w", reference, err)
	}
	return plain, nil
}
