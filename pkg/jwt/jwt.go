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
	now := time.Now()
	return s.signClaims(jwt.MapClaims{
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(duration).Unix(),
		"sub": subject,
	})
}

func (s *Signer) GenerateWithClaims(subject string, duration time.Duration, claims map[string]any) (string, error) {
	now := time.Now()
	tokenClaims := jwt.MapClaims{
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(duration).Unix(),
		"sub": subject,
	}

	for key, value := range claims {
		tokenClaims[key] = value
	}

	return s.signClaims(tokenClaims)
}

func (s *Signer) signClaims(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": claims["iat"],
		"nbf": claims["nbf"],
		"exp": claims["exp"],
		"sub": claims["sub"],
	})
	for key, value := range claims {
		token.Claims.(jwt.MapClaims)[key] = value
	}

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
