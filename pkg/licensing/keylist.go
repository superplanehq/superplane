package licensing

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
)

const (
	KeyListType     = "superplane-license-keys+jwt"
	KeyListAudience = "superplane-license-keys"

	// MaxKeyListBytes bounds every key list input before any decoding.
	MaxKeyListBytes = 64 * 1024
)

// ErrInvalidKeyList is returned for every key list that SuperPlane must not
// trust. It does not say why, so that callers cannot leak details.
var ErrInvalidKeyList = errors.New("invalid license key list")

// KeyList is a verified, root-signed list of license signing keys.
type KeyList struct {
	Version  int64
	IssuedAt time.Time
	Keys     *KeySet
	Document string
}

type keyListClaims struct {
	Issuer   *string `json:"iss"`
	Audience *string `json:"aud"`
	Version  *int64  `json:"version"`
	IssuedAt *int64  `json:"iat"`
	Keys     *[]jwk  `json:"keys"`
}

// VerifyKeyList verifies a key list signed by one of the root keys. A signing
// key must never use the key ID of a root key.
func VerifyKeyList(raw []byte, roots *KeySet) (*KeyList, error) {
	if len(raw) > MaxKeyListBytes {
		return nil, ErrInvalidKeyList
	}

	document := string(bytes.TrimSpace(raw))
	segments := strings.Split(document, ".")
	if len(segments) != 3 || slices.Contains(segments, "") {
		return nil, ErrInvalidKeyList
	}

	header, err := decodeProtectedHeader(segments[0], KeyListType)
	if err != nil {
		return nil, ErrInvalidKeyList
	}

	rootKey, ok := roots.PublicKey(header.KeyID)
	if !ok {
		return nil, ErrInvalidKeyList
	}

	if err := verifySignature(rootKey, segments[0]+"."+segments[1], segments[2]); err != nil {
		return nil, ErrInvalidKeyList
	}

	payload, err := rawURLEncoding.DecodeString(segments[1])
	if err != nil {
		return nil, ErrInvalidKeyList
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var claims keyListClaims
	if err := decoder.Decode(&claims); err != nil {
		return nil, ErrInvalidKeyList
	}

	if claims.Issuer == nil || *claims.Issuer != ExpectedIssuer ||
		claims.Audience == nil || *claims.Audience != KeyListAudience ||
		claims.Version == nil || *claims.Version <= 0 ||
		claims.IssuedAt == nil || *claims.IssuedAt <= 0 ||
		claims.Keys == nil || len(*claims.Keys) == 0 {
		return nil, ErrInvalidKeyList
	}

	keys, err := newKeySet(*claims.Keys)
	if err != nil {
		return nil, ErrInvalidKeyList
	}

	for _, keyID := range keys.KeyIDs() {
		if _, isRoot := roots.PublicKey(keyID); isRoot {
			return nil, ErrInvalidKeyList
		}
	}

	return &KeyList{
		Version:  *claims.Version,
		IssuedAt: time.Unix(*claims.IssuedAt, 0).UTC(),
		Keys:     keys,
		Document: document,
	}, nil
}
