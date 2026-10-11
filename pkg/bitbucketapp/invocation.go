package bitbucketapp

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v4"
)

const systemTokenHeader = "x-forge-oauth-system"

// Invocation is a verified Forge call to SuperPlane.
type Invocation struct {
	AppID              string
	InstallationID     string
	APIBaseURL         string
	WorkspaceUUID      string
	InstallerAccountID string
	EventType          string
	SystemToken        string
	SystemTokenExpires time.Time
}

type invocationClaims struct {
	jwtlib.RegisteredClaims
	App     invocationApp     `json:"app"`
	Context invocationContext `json:"context"`
}

type invocationApp struct {
	ID             string `json:"id"`
	InstallationID string `json:"installationId"`
	APIBaseURL     string `json:"apiBaseUrl"`
	Installation   struct {
		Contexts []struct {
			WorkspaceID string `json:"workspaceId"`
		} `json:"contexts"`
	} `json:"installation"`
}

type invocationContext struct {
	InstallContext string `json:"installContext"`
	CloudID        string `json:"cloudId"`
	WorkspaceID    string `json:"workspaceId"`
}

// ParseInvocation verifies a Forge Invocation Token and reads the system token
// expiry. The system token itself stays opaque except for the exp claim, which
// Forge documents as the way to know how long the token remains valid.
func ParseInvocation(fit, systemToken, expectedAppID string, keyfunc jwtlib.Keyfunc) (Invocation, error) {
	fit = strings.TrimSpace(fit)
	systemToken = strings.TrimSpace(systemToken)
	expectedAppID = strings.TrimSpace(expectedAppID)
	if fit == "" {
		return Invocation{}, errors.New("forge invocation token is required")
	}
	if systemToken == "" {
		return Invocation{}, errors.New("forge system token is required")
	}
	if expectedAppID == "" {
		return Invocation{}, errors.New("forge app id is required")
	}
	if keyfunc == nil {
		return Invocation{}, errors.New("forge verification key is required")
	}

	claims := &invocationClaims{}
	parsed, err := jwtlib.ParseWithClaims(fit, claims, keyfunc, jwtlib.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return Invocation{}, fmt.Errorf("verify forge invocation token: %w", err)
	}
	if !parsed.Valid {
		return Invocation{}, errors.New("forge invocation token is not valid")
	}
	if !audienceMatches(claims.Audience, expectedAppID) && claims.App.ID != expectedAppID {
		return Invocation{}, errors.New("forge invocation token is for a different app")
	}

	expires, err := systemTokenExpiry(systemToken)
	if err != nil {
		return Invocation{}, err
	}

	installationID := strings.TrimSpace(claims.App.InstallationID)
	if installationID == "" {
		return Invocation{}, errors.New("forge invocation token has no installation id")
	}

	return Invocation{
		AppID:              expectedAppID,
		InstallationID:     installationID,
		APIBaseURL:         strings.TrimSpace(claims.App.APIBaseURL),
		WorkspaceUUID:      workspaceUUID(claims),
		SystemToken:        systemToken,
		SystemTokenExpires: expires,
	}, nil
}

func audienceMatches(audience jwtlib.ClaimStrings, appID string) bool {
	for _, value := range audience {
		if value == appID {
			return true
		}
	}
	return false
}

func workspaceUUID(claims *invocationClaims) string {
	raw := strings.TrimSpace(claims.Context.WorkspaceID)
	if raw == "" {
		raw = strings.TrimSpace(claims.Context.InstallContext)
	}
	if raw == "" {
		for _, context := range claims.App.Installation.Contexts {
			if raw = strings.TrimSpace(context.WorkspaceID); raw != "" {
				break
			}
		}
	}
	if raw == "" {
		raw = strings.TrimSpace(claims.Context.CloudID)
	}
	const marker = "workspace/"
	if index := strings.LastIndex(raw, marker); index >= 0 {
		raw = raw[index+len(marker):]
	}
	return strings.Trim(strings.TrimSpace(raw), "{}")
}

func systemTokenExpiry(token string) (time.Time, error) {
	parser := jwtlib.NewParser()
	claims := jwtlib.MapClaims{}
	_, _, err := parser.ParseUnverified(token, claims)
	if err != nil {
		return time.Time{}, fmt.Errorf("read forge system token expiry: %w", err)
	}
	expires, ok := numericDate(claims["exp"])
	if !ok {
		return time.Time{}, errors.New("forge system token has no expiry")
	}
	return expires, nil
}

func numericDate(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case float64:
		return time.Unix(int64(typed), 0).UTC(), true
	case json.Number:
		seconds, err := typed.Int64()
		if err != nil {
			return time.Time{}, false
		}
		return time.Unix(seconds, 0).UTC(), true
	default:
		return time.Time{}, false
	}
}

// decodeJWKPart reads a JWK field. Forge publishes padded base64url.
func decodeJWKPart(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(value, "="))
}

// RSAPublicKey builds a verification key from a JWKS entry.
func RSAPublicKey(modulus, exponent string) (*rsa.PublicKey, error) {
	nBytes, err := decodeJWKPart(modulus)
	if err != nil {
		return nil, fmt.Errorf("decode forge key modulus: %w", err)
	}
	eBytes, err := decodeJWKPart(exponent)
	if err != nil {
		return nil, fmt.Errorf("decode forge key exponent: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	if !e.IsInt64() || e.Int64() <= 0 {
		return nil, errors.New("forge key exponent is invalid")
	}
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}
