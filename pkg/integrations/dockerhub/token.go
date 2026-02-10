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
	minRefreshInterval    = time.Minute
)

func refreshAccessToken(httpCtx core.HTTPContext, integration core.IntegrationContext) (*time.Duration, error) {
	username, err := integration.GetConfig("username")
	if err != nil {
		return nil, fmt.Errorf("failed to get username: %w", err)
	}

	accessToken, err := integration.GetConfig("accessToken")
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	token, refreshIn, err := createAccessToken(httpCtx, string(username), string(accessToken))
	if err != nil {
		return nil, err
	}

	if err := integration.SetSecret(accessTokenSecretName, []byte(token)); err != nil {
		return nil, err
	}

	return refreshIn, nil
}

type AccessTokenRequest struct {
	Identifier string `json:"identifier"`
	Secret     string `json:"secret"`
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func createAccessToken(httpCtx core.HTTPContext, identifier, secret string) (string, *time.Duration, error) {
	payload, err := json.Marshal(AccessTokenRequest{
		Identifier: identifier,
		Secret:     secret,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal access token request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, defaultBaseURL+authTokenEndpoint, bytes.NewReader(payload))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create access token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := httpCtx.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("access token request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read access token response: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", nil, fmt.Errorf("request failed with %d: %s", res.StatusCode, string(body))
	}

	var response AccessTokenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", nil, fmt.Errorf("failed to parse access token response: %w", err)
	}

	if response.AccessToken == "" {
		return "", nil, fmt.Errorf("access token response was empty")
	}

	expiresAt, err := parseJWTExpiry(response.AccessToken)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse access token expiry: %w", err)
	}

	//
	// We schedule the refresh for 1min before the expiration time.
	//
	interval := time.Until(expiresAt.Add(-time.Minute))
	if interval < minRefreshInterval {
		interval = minRefreshInterval
	}

	return response.AccessToken, &interval, nil
}

func parseJWTExpiry(token string) (*time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid JWT token")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse JWT payload: %w", err)
	}

	expValue, ok := payload["exp"]
	if !ok {
		return nil, fmt.Errorf("JWT exp not found")
	}

	expSeconds, err := parseJWTNumericClaim(expValue)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT exp: %w", err)
	}

	expiresAt := time.Unix(expSeconds, 0)
	return &expiresAt, nil
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
