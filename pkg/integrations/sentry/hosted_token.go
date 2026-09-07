package sentry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	jwtBearerGrantType = "urn:sentry:params:oauth:grant-type:jwt-bearer"
	tokenRefreshLeeway = 5 * time.Minute
)

var tokenNow = time.Now

type AppAuthorization struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    string `json:"expiresAt"`
}

func ExchangeSentryAppAuthorization(httpContext core.HTTPContext, app HostedApp, installationID, code string) (*AppAuthorization, error) {
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     app.ClientID,
		"client_secret": app.ClientSecret,
	}
	return postSentryAppAuthorization(httpContext, app.BaseURL, installationID, payload, "")
}

func RefreshSentryAppAuthorization(httpContext core.HTTPContext, app HostedApp, installationID string) (*AppAuthorization, error) {
	now := tokenNow()
	claims := jwt.RegisteredClaims{
		Issuer:    app.ClientID,
		Subject:   app.ClientID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
		ID:        uuid.NewString(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(app.ClientSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign Sentry refresh JWT: %w", err)
	}

	payload := map[string]string{
		"grant_type": jwtBearerGrantType,
	}
	return postSentryAppAuthorization(httpContext, app.BaseURL, installationID, payload, signed)
}

func storeInstallationAuthorization(integration core.IntegrationContext, authorization *AppAuthorization) error {
	if authorization == nil || strings.TrimSpace(authorization.Token) == "" {
		return fmt.Errorf("Sentry authorization token is missing")
	}

	if err := integration.SetSecret(secretInstallationToken, []byte(authorization.Token)); err != nil {
		return err
	}
	if authorization.RefreshToken != "" {
		if err := integration.SetSecret(secretRefreshToken, []byte(authorization.RefreshToken)); err != nil {
			return err
		}
	}
	if authorization.ExpiresAt != "" {
		if err := integration.SetSecret(secretTokenExpiresAt, []byte(authorization.ExpiresAt)); err != nil {
			return err
		}
	}

	return nil
}

func ensureInstallationToken(httpContext core.HTTPContext, integration core.IntegrationContext, app HostedApp, installationID string) (string, error) {
	token := secretValue(integration, secretInstallationToken)
	if token == "" {
		return "", fmt.Errorf("Sentry installation token is missing")
	}
	if installationID == "" {
		return token, nil
	}
	if !installationTokenNeedsRefresh(integration) {
		return token, nil
	}

	authorization, err := RefreshSentryAppAuthorization(httpContext, app, installationID)
	if err != nil {
		return "", fmt.Errorf("failed to refresh Sentry installation token: %w", err)
	}
	if err := storeInstallationAuthorization(integration, authorization); err != nil {
		return "", err
	}

	return authorization.Token, nil
}

func installationTokenNeedsRefresh(integration core.IntegrationContext) bool {
	raw := secretValue(integration, secretTokenExpiresAt)
	if raw == "" {
		return true
	}

	expiresAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return true
	}

	return !tokenNow().Add(tokenRefreshLeeway).Before(expiresAt)
}

func secretValue(integration core.IntegrationContext, name string) string {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return ""
	}

	for _, secret := range secrets {
		if secret.Name == name && len(secret.Value) > 0 {
			return string(secret.Value)
		}
	}

	return ""
}

func postSentryAppAuthorization(
	httpContext core.HTTPContext,
	baseURL, installationID string,
	payload map[string]string,
	bearerJWT string,
) (*AppAuthorization, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		strings.TrimRight(baseURL, "/")+"/api/0/sentry-app-installations/"+installationID+"/authorizations/",
		bytes.NewReader(encoded),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if bearerJWT != "" {
		req.Header.Set("Authorization", "Bearer "+bearerJWT)
	}

	resp, err := httpContext.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &apiError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	authorization := AppAuthorization{}
	if err := json.Unmarshal(body, &authorization); err != nil {
		return nil, err
	}
	if strings.TrimSpace(authorization.Token) == "" {
		return nil, fmt.Errorf("Sentry authorization response omitted token")
	}

	return &authorization, nil
}
