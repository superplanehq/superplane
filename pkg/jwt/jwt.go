package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const oktaOAuthStateSubject = "okta_oauth_state"
const oktaSAMLStateSubject = "okta_saml_state"

type Signer struct {
	Secret string
}

func NewSigner(secret string) *Signer {
	return &Signer{Secret: secret}
}

func (s *Signer) Generate(subject string, duration time.Duration) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(duration).Unix(),
		"sub": subject,
	})

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

// SignOktaOAuthState returns an opaque value for the OIDC `state` parameter (CSRF + context).
func (s *Signer) SignOktaOAuthState(orgID, redirect string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(ttl).Unix(),
		"sub": oktaOAuthStateSubject,
		"org": orgID,
		"rd":  redirect,
	})
	return token.SignedString([]byte(s.Secret))
}

// SignOktaSAMLState returns a signed RelayState JWT for SAML SP-initiated flow.
// It carries the SAML AuthnRequest ID (for replay prevention), the org ID, and optional redirect.
func (s *Signer) SignOktaSAMLState(orgID, requestID, redirect string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(ttl).Unix(),
		"sub": oktaSAMLStateSubject,
		"org": orgID,
		"rid": requestID,
		"rd":  redirect,
	})
	return token.SignedString([]byte(s.Secret))
}

// ParseOktaSAMLState validates the SAML RelayState JWT and returns org id, request id, and redirect path.
func (s *Signer) ParseOktaSAMLState(tokenString string) (orgID, requestID, redirect string, err error) {
	claims, err := s.ValidateAndGetClaims(tokenString)
	if err != nil {
		return "", "", "", err
	}
	if claims["sub"] != oktaSAMLStateSubject {
		return "", "", "", fmt.Errorf("invalid SAML state token")
	}
	org, _ := claims["org"].(string)
	if org == "" {
		return "", "", "", fmt.Errorf("invalid SAML state: missing org")
	}
	rid, _ := claims["rid"].(string)
	rd, _ := claims["rd"].(string)
	return org, rid, rd, nil
}

// ParseOktaOAuthState validates `state` from the OIDC callback and returns org id and redirect path.
func (s *Signer) ParseOktaOAuthState(tokenString string) (orgID, redirect string, err error) {
	claims, err := s.ValidateAndGetClaims(tokenString)
	if err != nil {
		return "", "", err
	}
	if claims["sub"] != oktaOAuthStateSubject {
		return "", "", fmt.Errorf("invalid state token")
	}
	org, _ := claims["org"].(string)
	if org == "" {
		return "", "", fmt.Errorf("invalid state org")
	}
	rd, _ := claims["rd"].(string)
	return org, rd, nil
}
