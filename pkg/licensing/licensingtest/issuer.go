// Package licensingtest signs licenses with ephemeral keys for tests. Keys
// exist only in memory for the duration of a test process.
package licensingtest

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/licensing"
)

var rawURL = base64.RawURLEncoding

type Issuer struct {
	KeyID      string
	privateKey *ecdsa.PrivateKey
}

func NewIssuer(keyID string) *Issuer {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("generate test signing key: %v", err))
	}

	return &Issuer{KeyID: keyID, privateKey: privateKey}
}

// JWKS returns the public JWKS document for the issuers.
func JWKS(issuers ...*Issuer) []byte {
	keys := make([]map[string]string, 0, len(issuers))
	for _, issuer := range issuers {
		publicKey, err := issuer.privateKey.PublicKey.Bytes()
		if err != nil {
			panic(fmt.Sprintf("encode test public key: %v", err))
		}

		keys = append(keys, map[string]string{
			"kty": "EC",
			"crv": "P-256",
			"use": "sig",
			"alg": "ES256",
			"kid": issuer.KeyID,
			"x":   rawURL.EncodeToString(publicKey[1:33]),
			"y":   rawURL.EncodeToString(publicKey[33:65]),
		})
	}

	return mustMarshal(map[string]any{"keys": keys})
}

func KeySet(issuers ...*Issuer) *licensing.KeySet {
	keySet, err := licensing.ParseKeySet(JWKS(issuers...))
	if err != nil {
		panic(fmt.Sprintf("parse test key set: %v", err))
	}

	return keySet
}

// Claims returns valid claims for a license that is active now and grants
// the features.
func Claims(features ...licensing.Feature) map[string]any {
	now := time.Now()
	keys := make([]string, 0, len(features))
	for _, feature := range features {
		keys = append(keys, string(feature))
	}

	return map[string]any{
		"iss":            licensing.ExpectedIssuer,
		"aud":            licensing.ExpectedAudience,
		"sub":            uuid.NewString(),
		"jti":            uuid.NewString(),
		"schema_version": 1,
		"edition":        "enterprise",
		"features":       keys,
		"iat":            now.Add(-time.Hour).Unix(),
		"nbf":            now.Add(-time.Hour).Unix(),
		"exp":            now.Add(365 * 24 * time.Hour).Unix(),
	}
}

func Header(keyID string) map[string]any {
	return map[string]any{
		"alg": licensing.SigningAlgorithm,
		"kid": keyID,
		"typ": licensing.TokenType,
	}
}

// Sign returns a compact JWS license for the claims.
func (i *Issuer) Sign(claims map[string]any) []byte {
	return i.SignWithHeader(Header(i.KeyID), claims)
}

func (i *Issuer) SignWithHeader(header map[string]any, claims map[string]any) []byte {
	signingInput := rawURL.EncodeToString(mustMarshal(header)) + "." + rawURL.EncodeToString(mustMarshal(claims))
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, i.privateKey, digest[:])
	if err != nil {
		panic(fmt.Sprintf("sign test license: %v", err))
	}

	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])
	return []byte(signingInput + "." + rawURL.EncodeToString(signature))
}

// License signs a valid license that grants the features.
func (i *Issuer) License(features ...licensing.Feature) []byte {
	return i.Sign(Claims(features...))
}

// KeyListClaims returns valid key list claims that trust the signers.
func KeyListClaims(version int64, signers ...*Issuer) map[string]any {
	var set struct {
		Keys []map[string]string `json:"keys"`
	}
	if err := json.Unmarshal(JWKS(signers...), &set); err != nil {
		panic(fmt.Sprintf("decode test key set: %v", err))
	}

	return map[string]any{
		"iss":     licensing.ExpectedIssuer,
		"aud":     licensing.KeyListAudience,
		"version": version,
		"iat":     time.Now().Unix(),
		"keys":    set.Keys,
	}
}

func KeyListHeader(keyID string) map[string]any {
	return map[string]any{
		"alg": licensing.SigningAlgorithm,
		"kid": keyID,
		"typ": licensing.KeyListType,
	}
}

// KeyList signs, as a root key, a key list that trusts the signers.
func (i *Issuer) KeyList(version int64, signers ...*Issuer) []byte {
	return i.SignWithHeader(KeyListHeader(i.KeyID), KeyListClaims(version, signers...))
}

// StaticSource is a read-only license source for tests.
type StaticSource struct {
	Raw []byte
}

func (s StaticSource) Kind() licensing.SourceKind {
	return licensing.SourceFile
}

func (s StaticSource) Read(context.Context) ([]byte, error) {
	return s.Raw, nil
}

// EnterpriseService returns a loaded license service with an active license
// that grants the features.
func EnterpriseService(features ...licensing.Feature) *licensing.Service {
	issuer := NewIssuer("licensingtest")
	service := licensing.NewService(
		licensing.NewVerifier(KeySet(issuer)),
		StaticSource{Raw: issuer.License(features...)},
	)

	if err := service.Refresh(context.Background()); err != nil {
		panic(fmt.Sprintf("load test license: %v", err))
	}

	return service
}

func mustMarshal(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("encode test value: %v", err))
	}

	return data
}
