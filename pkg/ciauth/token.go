package ciauth

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/oidc"
)

const ExecutionTokenAudience = "superplane-ci"

const (
	ClaimOrgID       = "org_id"
	ClaimCanvasID    = "canvas_id"
	ClaimNodeID      = "node_id"
	ClaimExecutionID = "execution_id"
	ClaimComponent   = "component"
)

type ExecutionTokenClaims struct {
	Subject     string
	Audience    string
	OrgID       string
	CanvasID    string
	NodeID      string
	ExecutionID string
	Component   string
	IssuedAt    int64
	ExpiresAt   int64

	// Additional holds any non-standard claims emitted by the triggering component.
	Additional map[string]string
}

type ExecutionTokenExpected struct {
	OrgID      string
	CanvasID   string
	NodeID     string
	Component  string
	Additional map[string]string
}

func ValidateToken(provider oidc.Provider, tokenString string) (ExecutionTokenClaims, error) {
	if provider == nil {
		return ExecutionTokenClaims{}, fmt.Errorf("OIDC provider is not configured")
	}

	rawClaims, err := provider.Validate(tokenString)
	if err != nil {
		return ExecutionTokenClaims{}, err
	}

	return ParseExecutionTokenClaims(rawClaims)
}

func ParseExecutionTokenClaims(raw map[string]any) (ExecutionTokenClaims, error) {
	claims := ExecutionTokenClaims{
		Subject:     stringClaim(raw, "sub"),
		Audience:    audienceClaim(raw),
		OrgID:       stringClaim(raw, ClaimOrgID),
		CanvasID:    stringClaim(raw, ClaimCanvasID),
		NodeID:      stringClaim(raw, ClaimNodeID),
		ExecutionID: stringClaim(raw, ClaimExecutionID),
		Component:   stringClaim(raw, ClaimComponent),
		IssuedAt:    int64Claim(raw, "iat"),
		ExpiresAt:   int64Claim(raw, "exp"),
		Additional:  additionalClaims(raw),
	}

	if claims.OrgID == "" || claims.CanvasID == "" || claims.NodeID == "" || claims.ExecutionID == "" {
		return ExecutionTokenClaims{}, fmt.Errorf("token is missing required execution claims")
	}

	if claims.Audience != ExecutionTokenAudience {
		return ExecutionTokenClaims{}, fmt.Errorf("token audience must be %q", ExecutionTokenAudience)
	}

	return claims, nil
}

func (expected ExecutionTokenExpected) Matches(claims ExecutionTokenClaims) error {
	if expected.OrgID != "" && expected.OrgID != claims.OrgID {
		return fmt.Errorf("org_id mismatch")
	}
	if expected.CanvasID != "" && expected.CanvasID != claims.CanvasID {
		return fmt.Errorf("canvas_id mismatch")
	}
	if expected.NodeID != "" && expected.NodeID != claims.NodeID {
		return fmt.Errorf("node_id mismatch")
	}
	if expected.Component != "" && expected.Component != claims.Component {
		return fmt.Errorf("component mismatch")
	}

	for key, value := range expected.Additional {
		if value == "" {
			continue
		}
		if claims.Additional[key] != value {
			return fmt.Errorf("%s mismatch", key)
		}
	}

	return nil
}

func additionalClaims(raw map[string]any) map[string]string {
	reserved := map[string]struct{}{
		"sub":              {},
		"aud":              {},
		"iss":              {},
		"iat":              {},
		"nbf":              {},
		"exp":              {},
		ClaimOrgID:         {},
		ClaimCanvasID:      {},
		ClaimNodeID:        {},
		ClaimExecutionID:   {},
		ClaimComponent:     {},
	}

	additional := make(map[string]string)
	for key, value := range raw {
		if _, ok := reserved[key]; ok {
			continue
		}
		if value == nil {
			continue
		}
		additional[key] = fmt.Sprint(value)
	}

	return additional
}

func stringClaim(raw map[string]any, key string) string {
	value, ok := raw[key]
	if !ok || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func int64Claim(raw map[string]any, key string) int64 {
	value, ok := raw[key]
	if !ok || value == nil {
		return 0
	}

	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		return 0
	}
}

func audienceClaim(raw map[string]any) string {
	value, ok := raw["aud"]
	if !ok || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		if len(typed) == 0 {
			return ""
		}
		return fmt.Sprint(typed[0])
	default:
		return fmt.Sprint(typed)
	}
}
