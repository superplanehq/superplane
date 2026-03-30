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
	Subject   string             `json:"sub"`
	Audience  string             `json:"aud"`
	ExpiresAt *gojwt.NumericDate `json:"exp,omitempty"`
	NotBefore *gojwt.NumericDate `json:"nbf,omitempty"`
	IssuedAt  *gojwt.NumericDate `json:"iat,omitempty"`
	TokenType string             `json:"token_type"`
	OrgID     string             `json:"org_id"`
	Purpose   string             `json:"purpose"`
	Scopes    []string           `json:"scopes"`
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

	scopes := normalizeScopedTokenValues(claims.Scopes)
	if len(scopes) == 0 {
		return "", fmt.Errorf("at least one scope is required")
	}

	now := time.Now()
	normalizedClaims := ScopedTokenClaims{
		Subject:   subject,
		Audience:  ScopedTokenAudience,
		ExpiresAt: gojwt.NewNumericDate(now.Add(duration)),
		NotBefore: gojwt.NewNumericDate(now),
		IssuedAt:  gojwt.NewNumericDate(now),
		TokenType: ScopedTokenType,
		OrgID:     orgID,
		Purpose:   purpose,
		Scopes:    scopes,
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

	scopes := normalizeScopedTokenValues(claims.Scopes)
	if len(scopes) == 0 {
		return nil, fmt.Errorf("at least one scope is required")
	}

	if len(PermissionsFromScopes(scopes)) == 0 {
		return nil, fmt.Errorf("at least one scope is required")
	}

	claims.Subject = subject
	claims.Audience = strings.TrimSpace(claims.Audience)
	claims.OrgID = orgID
	claims.Purpose = purpose
	claims.Scopes = scopes

	return claims, nil
}

func (c ScopedTokenClaims) Valid() error {
	return gojwt.RegisteredClaims{
		Subject:   c.Subject,
		ExpiresAt: c.ExpiresAt,
		NotBefore: c.NotBefore,
		IssuedAt:  c.IssuedAt,
	}.Valid()
}

func (c ScopedTokenClaims) VerifyAudience(cmp string, req bool) bool {
	if c.Audience == "" {
		return !req
	}

	return c.Audience == cmp
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

func ScopesFromPermissions(permissions []Permission) []string {
	normalized := normalizeScopedTokenPermissions(permissions)
	scopes := make([]string, 0, len(normalized))

	for _, permission := range normalized {
		scopePrefix := fmt.Sprintf("%s:%s", permission.ResourceType, permission.Action)
		if len(permission.Resources) == 0 {
			scopes = append(scopes, scopePrefix)
			continue
		}

		for _, resourceID := range permission.Resources {
			scopes = append(scopes, fmt.Sprintf("%s:%s", scopePrefix, resourceID))
		}
	}

	return normalizeScopedTokenValues(scopes)
}

func PermissionsFromScopes(scopes []string) []Permission {
	type permissionKey struct {
		resourceType string
		action       string
		global       bool
	}

	normalizedScopes := normalizeScopedTokenValues(scopes)
	permissions := make([]Permission, 0, len(normalizedScopes))
	indexByKey := make(map[permissionKey]int)

	for _, scope := range normalizedScopes {
		parts := strings.Split(scope, ":")
		if len(parts) != 2 && len(parts) != 3 {
			continue
		}

		resourceType := strings.TrimSpace(parts[0])
		action := strings.TrimSpace(parts[1])
		if resourceType == "" || action == "" {
			continue
		}

		if len(parts) == 2 {
			key := permissionKey{resourceType: resourceType, action: action, global: true}
			if _, exists := indexByKey[key]; exists {
				continue
			}

			indexByKey[key] = len(permissions)
			permissions = append(permissions, Permission{
				ResourceType: resourceType,
				Action:       action,
			})
			continue
		}

		resourceID := strings.TrimSpace(parts[2])
		if resourceID == "" {
			continue
		}

		key := permissionKey{resourceType: resourceType, action: action, global: false}
		if idx, exists := indexByKey[key]; exists {
			if !slices.Contains(permissions[idx].Resources, resourceID) {
				permissions[idx].Resources = append(permissions[idx].Resources, resourceID)
			}
			continue
		}

		indexByKey[key] = len(permissions)
		permissions = append(permissions, Permission{
			ResourceType: resourceType,
			Action:       action,
			Resources:    []string{resourceID},
		})
	}

	return normalizeScopedTokenPermissions(permissions)
}
