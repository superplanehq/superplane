package sentry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

type Auth struct {
	client core.HTTPContext
}

func NewAuth(client core.HTTPContext) *Auth {
	return &Auth{client: client}
}

type TokenResponse struct {
	AccessToken  string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    string `json:"expiresAt"`
}

type VerifyInstallationResponse struct {
	App struct {
		UUID string `json:"uuid"`
		Slug string `json:"slug"`
	} `json:"app"`
	Organization struct {
		Slug string `json:"slug"`
	} `json:"organization"`
	UUID string `json:"uuid"`
}

func (t *TokenResponse) GetExpiration() time.Duration {
	if t.ExpiresAt == "" {
		return time.Hour
	}

	expiresAt, err := time.Parse(time.RFC3339, t.ExpiresAt)
	if err != nil {
		return time.Hour
	}

	duration := time.Until(expiresAt) / 2
	if duration < time.Second {
		return time.Second
	}

	return duration
}

func (a *Auth) HandleCallback(req *http.Request, baseURL, clientID, clientSecret string) (*TokenResponse, string, error) {
	if errParam := req.URL.Query().Get("error"); errParam != "" {
		return nil, "", fmt.Errorf("OAuth error: %s", errParam)
	}

	code := req.URL.Query().Get("code")
	installationID := req.URL.Query().Get("installationId")
	if code == "" || installationID == "" {
		return nil, "", fmt.Errorf("missing code or installationId")
	}

	token, err := a.ExchangeCode(baseURL, clientID, clientSecret, installationID, code)
	if err != nil {
		return nil, "", err
	}

	return token, installationID, nil
}

func (a *Auth) ExchangeCode(baseURL, clientID, clientSecret, installationID, code string) (*TokenResponse, error) {
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     clientID,
		"client_secret": clientSecret,
	}

	return a.authorize(baseURL, installationID, payload)
}

func (a *Auth) RefreshToken(baseURL, clientID, clientSecret, installationID, refreshToken string) (*TokenResponse, error) {
	if installationID == "" {
		return nil, fmt.Errorf("installation ID is required")
	}

	payload := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     clientID,
		"client_secret": clientSecret,
	}

	return a.authorize(baseURL, installationID, payload)
}

func (a *Auth) authorize(baseURL, installationID string, payload map[string]string) (*TokenResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/0/sentry-app-installations/%s/authorizations/", baseURL, installationID),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: status %d, body: %s", resp.StatusCode, string(responseBody))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	tokenResponse := TokenResponse{}
	if err := json.Unmarshal(responseBody, &tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}

func (a *Auth) VerifyInstallation(baseURL, installationID, accessToken string) (*VerifyInstallationResponse, error) {
	body, err := json.Marshal(map[string]string{"status": "installed"})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/api/0/sentry-app-installations/%s/", baseURL, installationID),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("installation verification failed: status %d, body: %s", resp.StatusCode, string(responseBody))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := VerifyInstallationResponse{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
