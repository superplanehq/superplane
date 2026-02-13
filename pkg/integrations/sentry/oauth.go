package sentry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type authorizationResponse struct {
	ID           string `json:"id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    string `json:"expiresAt"`
}

func refreshSentryAuthorizationToken(
	ctx context.Context,
	httpCtx core.HTTPContext,
	baseURL string,
	installationID string,
	refreshToken string,
	clientID string,
	clientSecret string,
) (*authorizationResponse, error) {
	URL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	basePath := strings.TrimRight(URL.Path, "/")
	URL.Path = basePath + fmt.Sprintf("/api/0/sentry-app-installations/%s/authorizations/", installationID)

	requestBody, err := json.Marshal(map[string]any{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     clientID,
		"client_secret": clientSecret,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL.String(), bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := httpCtx.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("sentry token refresh failed: %d %s", res.StatusCode, string(resBody))
	}

	var out authorizationResponse
	if err := json.Unmarshal(resBody, &out); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if strings.TrimSpace(out.Token) == "" {
		return nil, fmt.Errorf("missing token in response")
	}

	return &out, nil
}
