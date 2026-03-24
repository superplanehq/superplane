package jwt

import (
	"fmt"
	"slices"
	"strings"
	"time"

	gojwt "github.com/golang-jwt/jwt/v4"
)

const ScopedTokenType = "scoped"
const ScopedTokenAudience = "superplane_api"

type ScopedTokenClaims struct {
	gojwt.RegisteredClaims

	TokenType   string       `json:"token_type"`
	OrgID       string       `json:"org_id"`
	Purpose     string       `json:"purpose"`
	Permissions []Permission `json:"permissions"`
}

type Permission struct {
	ResourceType string   `json:"resourceType"`
	Action       string   `json:"action"`
	Resources    []string `json:"resources,omitempty"`
}

func (s *Signer) GenerateScopedToken(claims ScopedTokenClaims, duration time.Duration) (string, error) {
	subject := strings.TrimSpace(claims.Subject)
	if subject == "" {
		return "", fmt.Errorf("subject is required")
	}

	orgID := strings.TrimSpace(claims.OrgID)
	if orgID == "" {
		return "", fmt.Errorf("org_id is required")
	}

	purpose := strings.TrimSpace(claims.Purpose)
	if purpose == "" {
		return "", fmt.Errorf("purpose is required")
	}

	permissions := normalizeScopedTokenPermissions(claims.Permissions)
	if len(permissions) == 0 {
		return "", fmt.Errorf("at least one permission is required")
	}

	now := time.Now()
	normalizedClaims := ScopedTokenClaims{
		TokenType:   ScopedTokenType,
		OrgID:       orgID,
		Purpose:     purpose,
		Permissions: permissions,
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   subject,
			Audience:  gojwt.ClaimStrings{ScopedTokenAudience},
			IssuedAt:  gojwt.NewNumericDate(now),
			NotBefore: gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(duration)),
		},
	}

	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, normalizedClaims)
	return token.SignedString([]byte(s.Secret))
}

func (s *Signer) ValidateScopedToken(tokenString string) (*ScopedTokenClaims, error) {
	token, err := gojwt.ParseWithClaims(tokenString, &ScopedTokenClaims{}, func(token *gojwt.Token) (any, error) {
		if _, ok := token.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(s.Secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*ScopedTokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	if claims.TokenType != ScopedTokenType {
		return nil, fmt.Errorf("invalid token_type")
	}

	if !claims.VerifyAudience(ScopedTokenAudience, true) {
		return nil, fmt.Errorf("invalid audience")
	}

	subject := strings.TrimSpace(claims.Subject)
	if subject == "" {
		return nil, fmt.Errorf("subject is required")
	}

	orgID := strings.TrimSpace(claims.OrgID)
	if orgID == "" {
		return nil, fmt.Errorf("org_id is required")
	}

	purpose := strings.TrimSpace(claims.Purpose)
	if purpose == "" {
		return nil, fmt.Errorf("purpose is required")
	}

	permissions := normalizeScopedTokenPermissions(claims.Permissions)
	if len(permissions) == 0 {
		return nil, fmt.Errorf("at least one permission is required")
	}

	claims.Subject = subject
	claims.OrgID = orgID
	claims.Purpose = purpose
	claims.Permissions = permissions

	return claims, nil
}

func normalizeScopedTokenValues(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if slices.Contains(normalized, trimmed) {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	return normalized
}

func normalizeScopedTokenPermissions(permissions []Permission) []Permission {
	normalized := make([]Permission, 0, len(permissions))
	for _, permission := range permissions {
		resourceType := strings.TrimSpace(permission.ResourceType)
		action := strings.TrimSpace(permission.Action)
		if resourceType == "" || action == "" {
			continue
		}

		normalizedPermission := Permission{
			ResourceType: resourceType,
			Action:       action,
		}

		resources := normalizeScopedTokenValues(permission.Resources)
		if len(resources) > 0 {
			normalizedPermission.Resources = resources
		}

		if slices.ContainsFunc(normalized, func(existing Permission) bool {
			return existing.ResourceType == normalizedPermission.ResourceType &&
				existing.Action == normalizedPermission.Action &&
				slices.Equal(existing.Resources, normalizedPermission.Resources)
		}) {
			continue
		}

		normalized = append(normalized, normalizedPermission)
	}

	return normalized
}
