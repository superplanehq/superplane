package dockerhub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	accessTokenSecretName      = "accessTokenJwt"
	accessTokenRefreshInterval = 9 * time.Minute
	authTokenEndpoint          = "/v2/auth/token"
)

type accessTokenRequest struct {
	Identifier string `json:"identifier"`
	Secret     string `json:"secret"`
}

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func refreshAccessToken(httpCtx core.HTTPContext, integration core.IntegrationContext, config Configuration) error {
	username := strings.TrimSpace(config.Username)
	if username == "" {
		return fmt.Errorf("username is required")
	}

	pat := strings.TrimSpace(config.AccessToken)
	if pat == "" {
		return fmt.Errorf("accessToken is required")
	}

	token, err := createAccessToken(httpCtx, username, pat)
	if err != nil {
		return err
	}

	return integration.SetSecret(accessTokenSecretName, []byte(token))
}

func createAccessToken(httpCtx core.HTTPContext, identifier, secret string) (string, error) {
	payload, err := json.Marshal(accessTokenRequest{
		Identifier: identifier,
		Secret:     secret,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal access token request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, defaultBaseURL+authTokenEndpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create access token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := httpCtx.Do(req)
	if err != nil {
		return "", fmt.Errorf("access token request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read access token response: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", &APIError{StatusCode: res.StatusCode, Body: string(body)}
	}

	var response accessTokenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse access token response: %w", err)
	}

	if response.AccessToken == "" {
		return "", fmt.Errorf("access token response was empty")
	}

	return response.AccessToken, nil
}
