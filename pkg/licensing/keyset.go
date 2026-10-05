package licensing

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
)

var keyIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

// KeySet holds the reviewed public keys that SuperPlane trusts to verify
// license signatures. It is built from bundled data only and never from the
// network.
type KeySet struct {
	keys map[string]*ecdsa.PublicKey
}

type jwkSet struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	KeyType   string `json:"kty"`
	Curve     string `json:"crv"`
	Use       string `json:"use"`
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	X         string `json:"x"`
	Y         string `json:"y"`
}

// ParseKeySet parses a JWKS document and rejects any key that is not an
// ES256 P-256 signing key with a unique, well-formed key ID.
func ParseKeySet(data []byte) (*KeySet, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var set jwkSet
	if err := decoder.Decode(&set); err != nil {
		return nil, fmt.Errorf("decode key set: %w", err)
	}

	if len(set.Keys) == 0 {
		return nil, errors.New("key set has no keys")
	}

	keys := make(map[string]*ecdsa.PublicKey, len(set.Keys))
	for _, key := range set.Keys {
		publicKey, err := key.publicKey()
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", key.KeyID, err)
		}

		if _, duplicate := keys[key.KeyID]; duplicate {
			return nil, fmt.Errorf("duplicate key ID %q", key.KeyID)
		}

		keys[key.KeyID] = publicKey
	}

	return &KeySet{keys: keys}, nil
}

// Merge returns a key set with the keys of both sets. Duplicate key IDs are
// an error because a key ID must never be reused.
func (k *KeySet) Merge(other *KeySet) (*KeySet, error) {
	keys := make(map[string]*ecdsa.PublicKey, len(k.keys)+len(other.keys))
	for keyID, publicKey := range k.keys {
		keys[keyID] = publicKey
	}

	for keyID, publicKey := range other.keys {
		if _, duplicate := keys[keyID]; duplicate {
			return nil, fmt.Errorf("duplicate key ID %q", keyID)
		}
		keys[keyID] = publicKey
	}

	return &KeySet{keys: keys}, nil
}

func (k *KeySet) KeyIDs() []string {
	keyIDs := make([]string, 0, len(k.keys))
	for keyID := range k.keys {
		keyIDs = append(keyIDs, keyID)
	}

	slices.Sort(keyIDs)
	return keyIDs
}

func (k *KeySet) lookup(keyID string) (*ecdsa.PublicKey, bool) {
	publicKey, ok := k.keys[keyID]
	return publicKey, ok
}

func (j jwk) publicKey() (*ecdsa.PublicKey, error) {
	if !keyIDPattern.MatchString(j.KeyID) {
		return nil, errors.New("invalid key ID")
	}

	if j.KeyType != "EC" || j.Curve != "P-256" || j.Use != "sig" || j.Algorithm != "ES256" {
		return nil, errors.New("key must be an EC P-256 ES256 signing key")
	}

	x, err := decodeCoordinate(j.X)
	if err != nil {
		return nil, fmt.Errorf("x coordinate: %w", err)
	}

	y, err := decodeCoordinate(j.Y)
	if err != nil {
		return nil, fmt.Errorf("y coordinate: %w", err)
	}

	point := append([]byte{0x04}, append(x, y...)...)
	publicKey, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), point)
	if err != nil {
		return nil, errors.New("coordinates are not a valid P-256 point")
	}

	return publicKey, nil
}

func decodeCoordinate(value string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil {
		return nil, errors.New("invalid base64url encoding")
	}

	if len(decoded) != 32 {
		return nil, errors.New("coordinate must be 32 bytes")
	}

	return decoded, nil
}
