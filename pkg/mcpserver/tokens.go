package mcpserver

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/jwt"
)

const AccessTokenTTL = time.Hour
const ConsentTTL = 10 * time.Minute

type AccessClaims struct {
	UserID    uuid.UUID
	OrgID     uuid.UUID
	FactoryID uuid.UUID
	Resource  string
	Scopes    []string
}

func MintAccessToken(signer *jwt.Signer, claims AccessClaims, ttl time.Duration) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("jwt signer is required")
	}
	if claims.UserID == uuid.Nil || claims.OrgID == uuid.Nil || claims.FactoryID == uuid.Nil {
		return "", fmt.Errorf("access token scope is incomplete")
	}
	resource := strings.TrimSpace(claims.Resource)
	if resource == "" {
		return "", fmt.Errorf("resource is required")
	}
	if ttl <= 0 {
		ttl = AccessTokenTTL
	}
	return signer.GenerateWithClaims(ttl, map[string]string{
		"purpose":    AccessTokenPurpose,
		"sub":        claims.UserID.String(),
		"org_id":     claims.OrgID.String(),
		"factory_id": claims.FactoryID.String(),
		"aud":        resource,
		"scope":      strings.Join(normalizeScopes(claims.Scopes), " "),
	})
}

func ParseAccessToken(signer *jwt.Signer, token, resource string) (*AccessClaims, error) {
	if signer == nil {
		return nil, fmt.Errorf("jwt signer is required")
	}
	claims, err := signer.ValidateAndGetClaims(token)
	if err != nil {
		return nil, err
	}
	purpose, _ := claims["purpose"].(string)
	if purpose != AccessTokenPurpose {
		return nil, fmt.Errorf("invalid access token purpose")
	}
	audience, _ := claims["aud"].(string)
	if strings.TrimSpace(audience) != strings.TrimSpace(resource) {
		return nil, fmt.Errorf("invalid access token audience")
	}
	userID, err := parseClaimUUID(claims, "sub")
	if err != nil {
		return nil, err
	}
	orgID, err := parseClaimUUID(claims, "org_id")
	if err != nil {
		return nil, err
	}
	factoryID, err := parseClaimUUID(claims, "factory_id")
	if err != nil {
		return nil, err
	}
	scope, _ := claims["scope"].(string)
	return &AccessClaims{
		UserID:    userID,
		OrgID:     orgID,
		FactoryID: factoryID,
		Resource:  audience,
		Scopes:    strings.Fields(scope),
	}, nil
}

func (c AccessClaims) HasScope(scope string) bool {
	return slices.Contains(c.Scopes, strings.TrimSpace(scope))
}

type ConsentClaims struct {
	ClientID            string
	RedirectURI         string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Resource            string
	ResponseType        string
}

func MintConsentToken(signer *jwt.Signer, claims ConsentClaims) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("jwt signer is required")
	}
	return signer.GenerateWithClaims(ConsentTTL, map[string]string{
		"purpose":               ConsentPurpose,
		"client_id":             claims.ClientID,
		"redirect_uri":          claims.RedirectURI,
		"state":                 claims.State,
		"code_challenge":        claims.CodeChallenge,
		"code_challenge_method": claims.CodeChallengeMethod,
		"resource":              claims.Resource,
		"response_type":         claims.ResponseType,
	})
}

func ParseConsentToken(signer *jwt.Signer, token string) (*ConsentClaims, error) {
	if signer == nil {
		return nil, fmt.Errorf("jwt signer is required")
	}
	claims, err := signer.ValidateAndGetClaims(token)
	if err != nil {
		return nil, err
	}
	purpose, _ := claims["purpose"].(string)
	if purpose != ConsentPurpose {
		return nil, fmt.Errorf("invalid consent token purpose")
	}
	return &ConsentClaims{
		ClientID:            stringClaim(claims, "client_id"),
		RedirectURI:         stringClaim(claims, "redirect_uri"),
		State:               stringClaim(claims, "state"),
		CodeChallenge:       stringClaim(claims, "code_challenge"),
		CodeChallengeMethod: stringClaim(claims, "code_challenge_method"),
		Resource:            stringClaim(claims, "resource"),
		ResponseType:        stringClaim(claims, "response_type"),
	}, nil
}

func parseClaimUUID(claims map[string]any, key string) (uuid.UUID, error) {
	raw, _ := claims[key].(string)
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, fmt.Errorf("invalid %s", key)
	}
	return id, nil
}

func stringClaim(claims map[string]any, key string) string {
	value, _ := claims[key].(string)
	return strings.TrimSpace(value)
}

func normalizeScopes(scopes []string) []string {
	out := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed == "" || slices.Contains(out, trimmed) {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
