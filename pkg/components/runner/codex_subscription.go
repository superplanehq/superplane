package runner

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/jwt"
)

const (
	CodexSubscriptionTokenPurpose = "codex_subscription"
	EnvCodexSubscriptionToken     = "SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN"
	EnvCodexRefreshTokenURL       = "CODEX_REFRESH_TOKEN_URL_OVERRIDE"
	CodexSubscriptionRefreshPath  = "/api/v1/runner/codex-subscription/refresh"
	CodexOAuthClientID            = "app_EMoamEEZ73f0CkXaXp7hrann"
	CodexOAuthTokenURL            = "https://auth.openai.com/oauth/token"
	CodexAccessRefreshWindow      = 10 * time.Minute
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
	if email := codexAccountEmail(document); email != "" {
		return email
	}
	for _, key := range []string{"account_id", "accountId"} {
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

func codexAccountEmail(document map[string]any) string {
	email := codexBundleString(document, "email")
	if email == "" {
		parts := strings.Split(codexBundleString(document, "id_token"), ".")
		if len(parts) != 3 {
			return ""
		}
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return ""
		}
		var claims struct {
			Email string `json:"email"`
		}
		if json.Unmarshal(payload, &claims) != nil {
			return ""
		}
		email = strings.TrimSpace(claims.Email)
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 {
		return ""
	}
	return email
}

func looksLikeCodexToken(value string) bool {
	return len(value) > 40 || strings.HasPrefix(value, "eyJ")
}

// CodexRunnerAuthBundle copies a stored auth bundle for one runner.
// The OpenAI refresh token is replaced by that runner's lease secret.
func CodexRunnerAuthBundle(raw []byte, leaseSecret string, now time.Time) ([]byte, error) {
	document, err := decodeCodexAuthDocument(raw)
	if err != nil {
		return nil, err
	}
	previous := codexBundleString(document, "refresh_token")
	replaceCodexAuthValue(document, "refresh_token", leaseSecret)
	document["last_refresh"] = now.UTC().Format(time.RFC3339Nano)
	encoded, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("chatgpt subscription credentials are invalid")
	}
	if previous != "" && previous != leaseSecret && strings.Contains(string(encoded), previous) {
		return nil, fmt.Errorf("chatgpt subscription credentials are invalid")
	}
	return encoded, nil
}

// ReplaceCodexAuthTokens writes a refreshed token set into a stored auth bundle.
func ReplaceCodexAuthTokens(raw []byte, accessToken, refreshToken, idToken string) ([]byte, error) {
	document, err := decodeCodexAuthDocument(raw)
	if err != nil {
		return nil, err
	}
	if accessToken != "" {
		replaceCodexAuthValue(document, "access_token", accessToken)
	}
	if refreshToken != "" {
		replaceCodexAuthValue(document, "refresh_token", refreshToken)
	}
	if idToken != "" {
		replaceCodexAuthValue(document, "id_token", idToken)
	}
	document["last_refresh"] = time.Now().UTC().Format(time.RFC3339Nano)
	return json.Marshal(document)
}

// CodexAccessTokenExpiry reads the exp claim from an access token.
func CodexAccessTokenExpiry(accessToken string) (time.Time, bool) {
	parts := strings.Split(accessToken, ".")
	if len(parts) < 2 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, false
	}
	switch exp := claims["exp"].(type) {
	case float64:
		return time.Unix(int64(exp), 0).UTC(), true
	default:
		return time.Time{}, false
	}
}

// CodexAuthToken returns one string field from a stored auth bundle.
func CodexAuthToken(raw []byte, name string) string {
	document, err := decodeCodexAuthDocument(raw)
	if err != nil {
		return ""
	}
	return codexBundleString(document, name)
}

func decodeCodexAuthDocument(raw []byte) (map[string]any, error) {
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil || document == nil {
		return nil, fmt.Errorf("chatgpt subscription credentials are invalid")
	}
	return document, nil
}

func replaceCodexAuthValue(value any, name, replacement string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == name {
				typed[key] = replacement
				continue
			}
			replaceCodexAuthValue(child, name, replacement)
		}
	case []any:
		for _, child := range typed {
			replaceCodexAuthValue(child, name, replacement)
		}
	}
}
