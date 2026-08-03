package runnerregistrationtoken

import (
	"fmt"
	"strings"
	"time"

	gojwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

const (
	Purpose  = "runner_registration"
	Audience = "task_broker"
)

// Claims are embedded in a single-use registration JWT minted by fleet-manager
// (or any holder of the broker HMAC secret). The broker validates the signature
// and records JTI on first successful registration so the token cannot be reused.
type Claims struct {
	FleetID string `json:"fleet_id"`
	Purpose string `json:"purpose"`
	gojwt.RegisteredClaims
}

// Mint creates a signed registration JWT for fleetID that expires at expiresAt.
// Each call produces a unique JTI (single-use at the broker).
func Mint(fleetID, secret string, expiresAt time.Time) (string, error) {
	fleetID = strings.TrimSpace(fleetID)
	secret = strings.TrimSpace(secret)
	if fleetID == "" {
		return "", fmt.Errorf("fleet id is empty")
	}
	if secret == "" {
		return "", fmt.Errorf("jwt secret is empty")
	}
	now := time.Now().UTC()
	if !expiresAt.After(now) {
		return "", fmt.Errorf("expiresAt must be in the future")
	}
	claims := Claims{
		FleetID: fleetID,
		Purpose: Purpose,
		RegisteredClaims: gojwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Audience:  gojwt.ClaimStrings{Audience},
			ExpiresAt: gojwt.NewNumericDate(expiresAt.UTC()),
			IssuedAt:  gojwt.NewNumericDate(now),
			NotBefore: gojwt.NewNumericDate(now),
		},
	}
	return gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// Validate parses and checks a registration JWT. On success it returns the claims
// (including JTI). Callers must enforce single-use by recording claims.ID.
func Validate(tokenString, wantFleetID, secret string) (*Claims, error) {
	wantFleetID = strings.TrimSpace(wantFleetID)
	secret = strings.TrimSpace(secret)
	if wantFleetID == "" {
		return nil, fmt.Errorf("fleet id is empty")
	}
	if secret == "" {
		return nil, fmt.Errorf("jwt secret is empty")
	}

	claims := &Claims{}
	token, err := gojwt.ParseWithClaims(tokenString, claims, func(token *gojwt.Token) (any, error) {
		if _, ok := token.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.Purpose != Purpose {
		return nil, fmt.Errorf("invalid purpose")
	}
	if !claims.VerifyAudience(Audience, true) {
		return nil, fmt.Errorf("invalid audience")
	}
	if strings.TrimSpace(claims.FleetID) != wantFleetID {
		return nil, fmt.Errorf("fleet id mismatch")
	}
	if strings.TrimSpace(claims.ID) == "" {
		return nil, fmt.Errorf("jti required")
	}
	return claims, nil
}
