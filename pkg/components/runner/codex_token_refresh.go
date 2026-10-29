package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// CodexTokenSet is one ChatGPT OAuth token response.
type CodexTokenSet struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
}

// CodexTokenRefreshError is a failure from the ChatGPT token endpoint.
type CodexTokenRefreshError struct {
	Permanent bool
	Code      string
}

func (e *CodexTokenRefreshError) Error() string {
	if e == nil {
		return "chatgpt subscription refresh failed"
	}
	if e.Code != "" {
		return e.Code
	}
	if e.Permanent {
		return "chatgpt subscription refresh was rejected"
	}
	return "chatgpt subscription refresh failed"
}

// RefreshCodexTokens exchanges the stored refresh token with OpenAI.
func RefreshCodexTokens(ctx context.Context, client *http.Client, refreshToken string) (CodexTokenSet, error) {
	if client == nil {
		client = http.DefaultClient
	}
	body, err := json.Marshal(map[string]string{
		"client_id":     CodexOAuthClientID,
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	})
	if err != nil {
		return CodexTokenSet{}, err
	}
	ctx, cancel := codexRefreshContext(ctx)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, CodexOAuthTokenURL, bytes.NewReader(body))
	if err != nil {
		return CodexTokenSet{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return CodexTokenSet{}, &CodexTokenRefreshError{Permanent: false}
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxCodexAuthBundleBytes))
	if err != nil {
		return CodexTokenSet{}, &CodexTokenRefreshError{Permanent: false}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return CodexTokenSet{}, classifyCodexRefreshFailure(response.StatusCode, payload)
	}
	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil || strings.TrimSpace(parsed.AccessToken) == "" {
		return CodexTokenSet{}, &CodexTokenRefreshError{Permanent: false}
	}
	return CodexTokenSet{
		AccessToken:  parsed.AccessToken,
		RefreshToken: parsed.RefreshToken,
		IDToken:      parsed.IDToken,
	}, nil
}

func classifyCodexRefreshFailure(status int, body []byte) error {
	code := codexRefreshErrorCode(body)
	permanent := status == http.StatusUnauthorized || permanentCodexRefreshCode(code)
	if status >= 500 && !permanentCodexRefreshCode(code) {
		permanent = false
	}
	return &CodexTokenRefreshError{Permanent: permanent, Code: code}
}

func permanentCodexRefreshCode(code string) bool {
	switch strings.ToLower(code) {
	case "refresh_token_expired", "refresh_token_reused", "refresh_token_invalidated":
		return true
	default:
		return false
	}
}

func codexRefreshErrorCode(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if code, ok := payload["code"].(string); ok {
		return code
	}
	switch nested := payload["error"].(type) {
	case string:
		return nested
	case map[string]any:
		if code, ok := nested["code"].(string); ok {
			return code
		}
	}
	return ""
}

// CodexAccessTokenNeedsRefresh reports whether a runner may ask Codex to refresh soon.
func CodexAccessTokenNeedsRefresh(accessToken string, now time.Time) bool {
	expires, ok := CodexAccessTokenExpiry(accessToken)
	if !ok {
		return true
	}
	return !expires.After(now.Add(CodexAccessRefreshWindow))
}

// Keep the refresh call bounded when the caller has no deadline.
func codexRefreshContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, 15*time.Second)
}
