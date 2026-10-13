package licensing

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	RevocationListType     = "superplane-license-revocations+jwt"
	RevocationListAudience = "superplane-license-revocations"

	// MaxRevocationListBytes bounds every revocation list before decoding.
	MaxRevocationListBytes = 256 * 1024
)

// ErrInvalidRevocationList is returned for every revocation list that
// SuperPlane must not trust. It does not say why.
var ErrInvalidRevocationList = errors.New("invalid license revocation list")

// ErrRevocationListDowngrade is returned for a list older than the trusted
// one. Accepting it could drop a revocation the issuer already published.
var ErrRevocationListDowngrade = errors.New("license revocation list is older than the trusted list")

// RevocationList is a verified, root-signed set of revoked license ids.
type RevocationList struct {
	Version  int64
	IssuedAt time.Time
	ids      map[uuid.UUID]struct{}
	Document string
}

// Contains reports whether the list names the license id.
func (l *RevocationList) Contains(id uuid.UUID) bool {
	if l == nil {
		return false
	}

	_, ok := l.ids[id]
	return ok
}

type revocationListClaims struct {
	Issuer   *string   `json:"iss"`
	Audience *string   `json:"aud"`
	Version  *int64    `json:"version"`
	IssuedAt *int64    `json:"iat"`
	Revoked  *[]string `json:"revoked"`
}

// VerifyRevocationList verifies a revocation list signed by one of the root
// keys. It does not replace an older trusted list; callers compare versions.
func VerifyRevocationList(raw []byte, roots *KeySet) (*RevocationList, error) {
	if len(raw) > MaxRevocationListBytes {
		return nil, ErrInvalidRevocationList
	}

	document := string(bytes.TrimSpace(raw))
	segments := strings.Split(document, ".")
	if len(segments) != 3 || slices.Contains(segments, "") {
		return nil, ErrInvalidRevocationList
	}

	header, err := decodeProtectedHeader(segments[0], RevocationListType)
	if err != nil {
		return nil, ErrInvalidRevocationList
	}

	rootKey, ok := roots.PublicKey(header.KeyID)
	if !ok {
		return nil, ErrInvalidRevocationList
	}

	if err := verifySignature(rootKey, segments[0]+"."+segments[1], segments[2]); err != nil {
		return nil, ErrInvalidRevocationList
	}

	payload, err := rawURLEncoding.DecodeString(segments[1])
	if err != nil {
		return nil, ErrInvalidRevocationList
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var claims revocationListClaims
	if err := decoder.Decode(&claims); err != nil {
		return nil, ErrInvalidRevocationList
	}

	if claims.Issuer == nil || *claims.Issuer != ExpectedIssuer ||
		claims.Audience == nil || *claims.Audience != RevocationListAudience ||
		claims.Version == nil || *claims.Version <= 0 ||
		claims.IssuedAt == nil || *claims.IssuedAt <= 0 ||
		claims.Revoked == nil {
		return nil, ErrInvalidRevocationList
	}

	ids := make(map[uuid.UUID]struct{}, len(*claims.Revoked))
	for _, value := range *claims.Revoked {
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil || id.String() != value {
			return nil, ErrInvalidRevocationList
		}

		if _, duplicate := ids[id]; duplicate {
			return nil, ErrInvalidRevocationList
		}

		ids[id] = struct{}{}
	}

	return &RevocationList{
		Version:  *claims.Version,
		IssuedAt: time.Unix(*claims.IssuedAt, 0).UTC(),
		ids:      ids,
		Document: document,
	}, nil
}
