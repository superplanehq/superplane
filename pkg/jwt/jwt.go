package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
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
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(duration).Unix(),
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
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(s.Secret), nil
	})

	if err != nil {
		return err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		epochSeconds := time.Now().Unix()
		if !claims.VerifyExpiresAt(epochSeconds, true) {
			return errors.New("missing exp")
		}

		if !claims.VerifyNotBefore(epochSeconds, true) {
			return errors.New("missing nbf")
		}

		if claims["sub"] != subject {
			return errors.New("subject is invalid")
		}

		return nil
	}

	return errors.New("invalid token")
}

// ValidateAndGetClaims validates a JWT token and returns the claims
func (s *Signer) ValidateAndGetClaims(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
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

	// Check expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, fmt.Errorf("token expired")
		}
	}

	return claims, nil
}
