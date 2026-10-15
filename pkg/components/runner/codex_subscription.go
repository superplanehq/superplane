package runner

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/jwt"
)

const (
	CodexSubscriptionTokenPurpose = "codex_subscription"
	EnvCodexSubscriptionToken     = "SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN"
	maxCodexAuthBundleBytes       = 256 * 1024
)

// CodexSubscriptionScope is the only runner access to one connection reservation.
type CodexSubscriptionScope struct {
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	ConnectionID   uuid.UUID
	TaskID         uuid.UUID
	Generation     int64
}

func MintCodexSubscriptionToken(signer *jwt.Signer, scope CodexSubscriptionScope, ttl time.Duration) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("jwt signer is required")
	}
	if ttl <= 0 {
		ttl = time.Duration(DefaultExecutionTimeoutSeconds) * time.Second
	}
	if scope.OrganizationID == uuid.Nil || scope.FactoryID == uuid.Nil || scope.ConnectionID == uuid.Nil || scope.TaskID == uuid.Nil || scope.Generation < 1 {
		return "", fmt.Errorf("chatgpt subscription scope is incomplete")
	}
	return signer.GenerateWithClaims(ttl, map[string]string{
		"purpose":       CodexSubscriptionTokenPurpose,
		"org_id":        scope.OrganizationID.String(),
		"factory_id":    scope.FactoryID.String(),
		"connection_id": scope.ConnectionID.String(),
		"task_id":       scope.TaskID.String(),
		"generation":    fmt.Sprintf("%d", scope.Generation),
	})
}

func ParseCodexSubscriptionToken(signer *jwt.Signer, token string) (*CodexSubscriptionScope, error) {
	if signer == nil {
		return nil, fmt.Errorf("jwt signer is required")
	}
	claims, err := signer.ValidateAndGetClaims(token)
	if err != nil {
		return nil, err
	}
	purpose, _ := claims["purpose"].(string)
	if purpose != CodexSubscriptionTokenPurpose {
		return nil, fmt.Errorf("invalid chatgpt subscription token purpose")
	}
	scope := &CodexSubscriptionScope{}
	scope.OrganizationID, err = parsePlanningClaimUUID(claims, "org_id")
	if err != nil {
		return nil, err
	}
	scope.FactoryID, err = parsePlanningClaimUUID(claims, "factory_id")
	if err != nil {
		return nil, err
	}
	scope.ConnectionID, err = parsePlanningClaimUUID(claims, "connection_id")
	if err != nil {
		return nil, err
	}
	scope.TaskID, err = parsePlanningClaimUUID(claims, "task_id")
	if err != nil {
		return nil, err
	}
	generation, _ := claims["generation"].(string)
	if _, err = fmt.Sscan(strings.TrimSpace(generation), &scope.Generation); err != nil || scope.Generation < 1 {
		return nil, fmt.Errorf("invalid chatgpt subscription generation")
	}
	return scope, nil
}

// PrepareCodexAuthBundle checks that the JSON is a native Codex auth bundle
// and removes an API key before storage.
func PrepareCodexAuthBundle(raw []byte) ([]byte, string, error) {
	if len(raw) == 0 || len(raw) > maxCodexAuthBundleBytes {
		return nil, "", fmt.Errorf("chatgpt subscription credentials are invalid")
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil || document == nil {
		return nil, "", fmt.Errorf("chatgpt subscription credentials are invalid")
	}
	if strings.TrimSpace(codexBundleString(document, "access_token")) == "" ||
		strings.TrimSpace(codexBundleString(document, "refresh_token")) == "" ||
		strings.TrimSpace(codexBundleString(document, "id_token")) == "" {
		return nil, "", fmt.Errorf("chatgpt subscription credentials are invalid")
	}
	delete(document, "OPENAI_API_KEY")
	encoded, err := json.Marshal(document)
	if err != nil {
		return nil, "", fmt.Errorf("chatgpt subscription credentials are invalid")
	}
	return encoded, codexAccountLabel(document), nil
}

func codexBundleString(value any, name string) string {
	switch typed := value.(type) {
	case map[string]any:
		if raw, ok := typed[name].(string); ok && strings.TrimSpace(raw) != "" {
			return strings.TrimSpace(raw)
		}
		for _, child := range typed {
			if found := codexBundleString(child, name); found != "" {
				return found
			}
		}
	case []any:
		for _, child := range typed {
			if found := codexBundleString(child, name); found != "" {
				return found
			}
		}
	}
	return ""
}

func codexAccountLabel(document map[string]any) string {
	for _, key := range []string{"email", "account_id", "accountId"} {
		label := codexBundleString(document, key)
		if label == "" || looksLikeCodexToken(label) {
			continue
		}
		if len(label) > 80 {
			label = label[:80]
		}
		return label
	}
	return "ChatGPT account"
}

func looksLikeCodexToken(value string) bool {
	return len(value) > 40 || strings.HasPrefix(value, "eyJ")
}
