package oidc

import (
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
)

func PublicKeysFromJWKs(jwks []PublicJWK) (map[string]*rsa.PublicKey, error) {
	publicKeys := make(map[string]*rsa.PublicKey, len(jwks))

	for _, jwk := range jwks {
		if jwk.Kid == "" {
			return nil, fmt.Errorf("JWK is missing kid")
		}
		if _, exists := publicKeys[jwk.Kid]; exists {
			return nil, fmt.Errorf("duplicate JWK kid: %s", jwk.Kid)
		}

		publicKey, err := publicKeyFromJWK(jwk)
		if err != nil {
			return nil, fmt.Errorf("parse JWK %s: %w", jwk.Kid, err)
		}
		publicKeys[jwk.Kid] = publicKey
	}

	if len(publicKeys) == 0 {
		return nil, fmt.Errorf("no JWKs found")
	}

	return publicKeys, nil
}

func publicKeyFromJWK(jwk PublicJWK) (*rsa.PublicKey, error) {
	if jwk.Kty != "RSA" {
		return nil, fmt.Errorf("unsupported key type: %s", jwk.Kty)
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	if !e.IsInt64() {
		return nil, fmt.Errorf("invalid exponent")
	}

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
