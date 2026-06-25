package oidc

import (
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

func (s *RSAProvider) Validate(tokenString string) (map[string]any, error) {
	parser := jwt.Parser{
		ValidMethods: []string{jwt.SigningMethodRS256.Alg()},
	}

	token, err := parser.Parse(tokenString, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, fmt.Errorf("token is missing kid header")
		}

		publicKey, ok := s.publicKeys[kid]
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

	if issuer, _ := claims["iss"].(string); issuer != s.issuer {
		return nil, fmt.Errorf("invalid token issuer")
	}

	return map[string]any(claims), nil
}
