package mcpserver

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

const AccessTokenTTL = time.Hour
const ConsentTTL = 10 * time.Minute

type AccessClaims struct {
	UserID     uuid.UUID
	OrgID      uuid.UUID
	FactoryID  uuid.UUID
	FactoryIDs []uuid.UUID
	ClientID   string
	Resource   string
	Scopes     []string
}

func (c AccessClaims) WorkspaceIDs() []uuid.UUID {
	if len(c.FactoryIDs) > 0 {
		ids, err := models.NormalizeFactoryIDs(c.FactoryIDs)
		if err == nil {
			return ids
		}
	}
	if c.FactoryID == uuid.Nil {
		return nil
	}
	return []uuid.UUID{c.FactoryID}
}

func MintAccessToken(signer *jwt.Signer, claims AccessClaims, ttl time.Duration) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("jwt signer is required")
	}
	factoryIDs, err := claims.accessFactoryIDs()
	if err != nil || claims.UserID == uuid.Nil || claims.OrgID == uuid.Nil {
		return "", fmt.Errorf("access token scope is incomplete")
	}
	clientID := strings.TrimSpace(claims.ClientID)
	if clientID == "" {
		return "", fmt.Errorf("client_id is required")
	}
	resource := strings.TrimSpace(claims.Resource)
	if resource == "" {
		return "", fmt.Errorf("resource is required")
	}
	if ttl <= 0 {
		ttl = AccessTokenTTL
	}
	return signer.GenerateWithClaims(ttl, map[string]string{
		"purpose":     AccessTokenPurpose,
		"sub":         claims.UserID.String(),
		"org_id":      claims.OrgID.String(),
		"factory_id":  factoryIDs[0].String(),
		"factory_ids": formatFactoryIDsClaim(factoryIDs),
		"client_id":   clientID,
		"aud":         resource,
		"scope":       strings.Join(normalizeScopes(claims.Scopes), " "),
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
	factoryIDs, err := parseFactoryIDsClaim(stringClaim(claims, "factory_ids"), factoryID)
	if err != nil {
		return nil, err
	}
	clientID := stringClaim(claims, "client_id")
	if clientID == "" {
		return nil, fmt.Errorf("invalid client_id")
	}
	scope, _ := claims["scope"].(string)
	return &AccessClaims{
		UserID:     userID,
		OrgID:      orgID,
		FactoryID:  factoryIDs[0],
		FactoryIDs: factoryIDs,
		ClientID:   clientID,
		Resource:   audience,
		Scopes:     strings.Fields(scope),
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

func (c AccessClaims) accessFactoryIDs() ([]uuid.UUID, error) {
	if len(c.FactoryIDs) > 0 {
		return models.NormalizeFactoryIDs(c.FactoryIDs)
	}
	if c.FactoryID == uuid.Nil {
		return nil, fmt.Errorf("workspace is required")
	}
	return []uuid.UUID{c.FactoryID}, nil
}

func formatFactoryIDsClaim(ids []uuid.UUID) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = id.String()
	}
	return strings.Join(parts, ",")
}

func parseFactoryIDsClaim(raw string, fallback uuid.UUID) ([]uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if fallback == uuid.Nil {
			return nil, fmt.Errorf("invalid factory_id")
		}
		return []uuid.UUID{fallback}, nil
	}
	parts := strings.Split(raw, ",")
	ids := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		id, err := uuid.Parse(strings.TrimSpace(part))
		if err != nil || id == uuid.Nil {
			return nil, fmt.Errorf("invalid factory_ids")
		}
		ids = append(ids, id)
	}
	return models.NormalizeFactoryIDs(ids)
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
