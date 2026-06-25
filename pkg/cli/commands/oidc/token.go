package oidc

import (
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/golang-jwt/jwt/v4"
	spoidc "github.com/superplanehq/superplane/pkg/oidc"
)

func validateToken(tokenString, issuer string, publicKeys map[string]*rsa.PublicKey) (map[string]any, error) {
	parser := jwt.Parser{
		ValidMethods: []string{jwt.SigningMethodRS256.Alg()},
	}

	token, err := parser.Parse(tokenString, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, fmt.Errorf("token is missing kid header")
		}

		publicKey, ok := publicKeys[kid]
		if !ok {
			return nil, fmt.Errorf("unknown signing key")
		}

		return publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	if claimIssuer, _ := claims["iss"].(string); claimIssuer != issuer {
		return nil, fmt.Errorf("invalid token issuer")
	}

	return map[string]any(claims), nil
}

func publicKeysFromJWKs(jwks []spoidc.PublicJWK) (map[string]*rsa.PublicKey, error) {
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

func publicKeyFromJWK(jwk spoidc.PublicJWK) (*rsa.PublicKey, error) {
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
