package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Signer struct {
	Secret string
}

func NewSigner(secret string) *Signer {
	return &Signer{Secret: secret}
}

func (s *Signer) Generate(subject string, duration time.Duration) (string, error) {
	return s.GenerateWithClaims(duration, map[string]string{"sub": subject})
}

// GenerateWithClaims creates a JWT with the standard time claims plus any additional custom claims.
// Extra claims must not override the reserved time claims (iat, nbf, exp).
func (s *Signer) GenerateWithClaims(duration time.Duration, extraClaims map[string]string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": jwt.NewNumericDate(now),
		"nbf": jwt.NewNumericDate(now),
		"exp": jwt.NewNumericDate(now.Add(duration)),
	}

	for k, v := range extraClaims {
		switch k {
		case "iat", "nbf", "exp":
			return "", fmt.Errorf("cannot override reserved claim %q", k)
		}
		claims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(s.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *Signer) Validate(tokenString, subject string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(s.Secret), nil
	})
	if err != nil {
		return err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return errors.New("invalid token")
	}

	sub, _ := claims["sub"].(string)
	if sub != subject {
		return errors.New("subject is invalid")
	}

	return nil
}

// ValidateAndGetClaims validates a JWT token and returns the claims
func (s *Signer) ValidateAndGetClaims(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.Secret), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
