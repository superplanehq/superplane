package dockerhub

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	accessTokenSecretName = "accessTokenJwt"
	authTokenEndpoint     = "/v2/auth/token"
	tokenRefreshSkew      = time.Minute
	minRefreshInterval    = time.Minute
)

type accessTokenRequest struct {
	Identifier string `json:"identifier"`
	Secret     string `json:"secret"`
}

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func refreshAccessToken(httpCtx core.HTTPContext, integration core.IntegrationContext, config Configuration) (time.Time, error) {
	username := strings.TrimSpace(config.Username)
	if username == "" {
		return time.Time{}, fmt.Errorf("username is required")
	}

	pat := strings.TrimSpace(config.AccessToken)
	if pat == "" {
		return time.Time{}, fmt.Errorf("accessToken is required")
	}

	token, expiresAt, err := createAccessToken(httpCtx, username, pat)
	if err != nil {
		return time.Time{}, err
	}

	if err := integration.SetSecret(accessTokenSecretName, []byte(token)); err != nil {
		return time.Time{}, err
	}

	return expiresAt, nil
}

func createAccessToken(httpCtx core.HTTPContext, identifier, secret string) (string, time.Time, error) {
	payload, err := json.Marshal(accessTokenRequest{
		Identifier: identifier,
		Secret:     secret,
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to marshal access token request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, defaultBaseURL+authTokenEndpoint, bytes.NewReader(payload))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create access token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := httpCtx.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("access token request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to read access token response: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", time.Time{}, &APIError{StatusCode: res.StatusCode, Body: string(body)}
	}

	var response accessTokenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse access token response: %w", err)
	}

	if response.AccessToken == "" {
		return "", time.Time{}, fmt.Errorf("access token response was empty")
	}

	expiresAt, err := parseJWTExpiry(response.AccessToken)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse access token expiry: %w", err)
	}

	return response.AccessToken, expiresAt, nil
}

func parseJWTExpiry(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid JWT token")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return time.Time{}, fmt.Errorf("failed to parse JWT payload: %w", err)
	}

	expValue, ok := payload["exp"]
	if !ok {
		return time.Time{}, fmt.Errorf("JWT exp not found")
	}

	expSeconds, err := parseJWTNumericClaim(expValue)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid JWT exp: %w", err)
	}

	return time.Unix(expSeconds, 0), nil
}

func parseJWTNumericClaim(value any) (int64, error) {
	switch v := value.(type) {
	case float64:
		return int64(v), nil
	case json.Number:
		return v.Int64()
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case string:
		parsed, err := json.Number(v).Int64()
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported claim type %T", value)
	}
}

func scheduleAccessTokenRefresh(integration core.IntegrationContext, expiresAt time.Time) error {
	interval := time.Until(expiresAt.Add(-tokenRefreshSkew))
	if interval < minRefreshInterval {
		interval = minRefreshInterval
	}

	return integration.ScheduleActionCall("refreshAccessToken", map[string]any{}, interval)
}
