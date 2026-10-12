package licensing

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ExpectedIssuer   = "https://licensing.superplane.com"
	ExpectedAudience = "superplane-self-hosted"
	TokenType        = "superplane-license+jwt"
	SigningAlgorithm = "ES256"

	// MaxLicenseBytes bounds every license input before any decoding.
	MaxLicenseBytes = 16 * 1024

	ClockSkew = 5 * time.Minute

	supportedSchemaVersion = 1
)

var (
	rawURLEncoding    = base64.RawURLEncoding.Strict()
	featureKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
)

// Reason is a safe failure category. It never contains license content or
// cryptographic details.
type Reason string

const (
	ReasonMalformed            Reason = "malformed"
	ReasonUnsupportedAlgorithm Reason = "unsupported_algorithm"
	ReasonUnknownKey           Reason = "unknown_key"
	ReasonInvalidSignature     Reason = "invalid_signature"
	ReasonInvalidClaims        Reason = "invalid_claims"
	ReasonExpired              Reason = "expired"
	ReasonNotYetValid          Reason = "not_yet_valid"
	ReasonRevoked              Reason = "revoked"
	ReasonUnreadable           Reason = "unreadable"
)

type VerificationError struct {
	Reason Reason
}

func (e *VerificationError) Error() string {
	return "invalid license: " + string(e.Reason)
}

func invalid(reason Reason) error {
	return &VerificationError{Reason: reason}
}

// ReasonOf returns the failure category of a verification error.
func ReasonOf(err error) Reason {
	var verificationErr *VerificationError
	if errors.As(err, &verificationErr) {
		return verificationErr.Reason
	}

	return ReasonUnreadable
}

type protectedHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	Type      string `json:"typ"`
}

type claims struct {
	Issuer        *string   `json:"iss"`
	Audience      *string   `json:"aud"`
	Subject       *string   `json:"sub"`
	JWTID         *string   `json:"jti"`
	SchemaVersion *int      `json:"schema_version"`
	Edition       *string   `json:"edition"`
	Features      *[]string `json:"features"`
	IssuedAt      *int64    `json:"iat"`
	NotBefore     *int64    `json:"nbf"`
	ExpiresAt     *int64    `json:"exp"`
}

// Verifier checks license signatures and claims offline against trusted
// public keys. It does not check the validity window; callers use
// License.ValidityAt so an expired license can still be reported.
type Verifier struct {
	keys     PublicKeys
	issuer   string
	audience string
}

func NewVerifier(keys PublicKeys) *Verifier {
	return &Verifier{
		keys:     keys,
		issuer:   ExpectedIssuer,
		audience: ExpectedAudience,
	}
}

func (v *Verifier) Verify(raw []byte) (*License, error) {
	if len(raw) > MaxLicenseBytes {
		return nil, invalid(ReasonMalformed)
	}

	token := string(bytes.TrimSpace(raw))
	segments := strings.Split(token, ".")
	if len(segments) != 3 || slices.Contains(segments, "") {
		return nil, invalid(ReasonMalformed)
	}

	header, err := decodeHeader(segments[0])
	if err != nil {
		return nil, err
	}

	publicKey, ok := v.keys.PublicKey(header.KeyID)
	if !ok {
		return nil, invalid(ReasonUnknownKey)
	}

	if err := verifySignature(publicKey, segments[0]+"."+segments[1], segments[2]); err != nil {
		return nil, err
	}

	payload, err := rawURLEncoding.DecodeString(segments[1])
	if err != nil {
		return nil, invalid(ReasonMalformed)
	}

	var decoded claims
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, invalid(ReasonInvalidClaims)
	}

	license, err := v.licenseFromClaims(decoded)
	if err != nil {
		return nil, err
	}

	license.KeyID = header.KeyID
	return license, nil
}

func decodeHeader(segment string) (*protectedHeader, error) {
	return decodeProtectedHeader(segment, TokenType)
}

func decodeProtectedHeader(segment, tokenType string) (*protectedHeader, error) {
	data, err := rawURLEncoding.DecodeString(segment)
	if err != nil {
		return nil, invalid(ReasonMalformed)
	}

	// The issuer emits only alg, kid, and typ. Any other parameter, including
	// crit, jku, jwk, x5u, and x5c, is rejected.
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var header protectedHeader
	if err := decoder.Decode(&header); err != nil {
		return nil, invalid(ReasonMalformed)
	}

	if header.Algorithm != SigningAlgorithm {
		return nil, invalid(ReasonUnsupportedAlgorithm)
	}

	if header.Type != tokenType || header.KeyID == "" {
		return nil, invalid(ReasonMalformed)
	}

	return &header, nil
}

func verifySignature(publicKey *ecdsa.PublicKey, signingInput, encodedSignature string) error {
	signature, err := rawURLEncoding.DecodeString(encodedSignature)
	if err != nil || len(signature) != 64 {
		return invalid(ReasonInvalidSignature)
	}

	digest := sha256.Sum256([]byte(signingInput))
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	if !ecdsa.Verify(publicKey, digest[:], r, s) {
		return invalid(ReasonInvalidSignature)
	}

	return nil
}

func (v *Verifier) licenseFromClaims(c claims) (*License, error) {
	if c.Issuer == nil || *c.Issuer != v.issuer {
		return nil, invalid(ReasonInvalidClaims)
	}

	if c.Audience == nil || *c.Audience != v.audience {
		return nil, invalid(ReasonInvalidClaims)
	}

	if c.SchemaVersion == nil || *c.SchemaVersion != supportedSchemaVersion {
		return nil, invalid(ReasonInvalidClaims)
	}

	if c.Edition == nil || Edition(*c.Edition) != EditionEnterprise {
		return nil, invalid(ReasonInvalidClaims)
	}

	customerID, err := parseUUIDClaim(c.Subject)
	if err != nil {
		return nil, err
	}

	licenseID, err := parseUUIDClaim(c.JWTID)
	if err != nil {
		return nil, err
	}

	if c.IssuedAt == nil || c.NotBefore == nil || c.ExpiresAt == nil {
		return nil, invalid(ReasonInvalidClaims)
	}

	if *c.IssuedAt <= 0 || *c.NotBefore <= 0 || *c.ExpiresAt <= *c.NotBefore || *c.ExpiresAt <= *c.IssuedAt {
		return nil, invalid(ReasonInvalidClaims)
	}

	features, err := parseFeatures(c.Features)
	if err != nil {
		return nil, err
	}

	return &License{
		ID:         licenseID,
		CustomerID: customerID,
		Issuer:     *c.Issuer,
		Edition:    EditionEnterprise,
		Features:   features,
		IssuedAt:   time.Unix(*c.IssuedAt, 0).UTC(),
		ValidFrom:  time.Unix(*c.NotBefore, 0).UTC(),
		ExpiresAt:  time.Unix(*c.ExpiresAt, 0).UTC(),
	}, nil
}

func parseUUIDClaim(value *string) (uuid.UUID, error) {
	if value == nil {
		return uuid.Nil, invalid(ReasonInvalidClaims)
	}

	parsed, err := uuid.Parse(*value)
	if err != nil || parsed == uuid.Nil || parsed.String() != *value {
		return uuid.Nil, invalid(ReasonInvalidClaims)
	}

	return parsed, nil
}

func parseFeatures(values *[]string) ([]Feature, error) {
	if values == nil || len(*values) == 0 {
		return nil, invalid(ReasonInvalidClaims)
	}

	seen := make(map[string]bool, len(*values))
	features := []Feature{}
	for _, key := range *values {
		if !featureKeyPattern.MatchString(key) || seen[key] {
			return nil, invalid(ReasonInvalidClaims)
		}

		seen[key] = true
		if IsRecognizedFeature(key) {
			features = append(features, Feature(key))
		}
	}

	slices.Sort(features)
	return features, nil
}
