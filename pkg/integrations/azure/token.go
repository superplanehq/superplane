package azure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/oidc"
)

const (
	defaultScope = "https://management.azure.com/.default"
	tokenPath    = "/oauth2/v2.0/token"
)

// TokenResult holds the access token and its expiry from Azure AD.
type TokenResult struct {
	AccessToken string
	ExpiresIn   int // seconds
	ExpiresAt   time.Time
}

// ObtainToken exchanges an OIDC JWT (client assertion) for an Azure AD access token.
// It does not store anything; the caller (Sync) stores the token in integration secrets.
func ObtainToken(
	httpCtx core.HTTPContext,
	oidcProvider oidc.Provider,
	tenantID string,
	clientID string,
	integrationID string,
	requestLifetime time.Duration,
) (*TokenResult, error) {
	tenantID = strings.TrimSpace(tenantID)
	clientID = strings.TrimSpace(clientID)
	if tenantID == "" || clientID == "" {
		return nil, fmt.Errorf("tenant ID and client ID are required")
	}

	subject := fmt.Sprintf("app-installation:%s", integrationID)
	audience := integrationID
	oidcToken, err := oidcProvider.Sign(subject, requestLifetime, audience, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to sign OIDC token: %w", err)
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s%s", tenantID, tokenPath)
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	form.Set("client_assertion", oidcToken)
	form.Set("scope", defaultScope)
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := httpCtx.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if err := parseTokenError(body); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("Azure AD token request failed with %d: %s", res.StatusCode, string(body))
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	if payload.AccessToken == "" {
		return nil, fmt.Errorf("Azure AD response missing access_token")
	}

	expiresIn := payload.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}

	return &TokenResult{
		AccessToken: payload.AccessToken,
		ExpiresIn:   expiresIn,
		ExpiresAt:   time.Now().Add(time.Duration(expiresIn) * time.Second),
	}, nil
}

func parseTokenError(body []byte) error {
	var payload struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	if payload.Error != "" || payload.ErrorDescription != "" {
		return fmt.Errorf("Azure AD: %s %s", strings.TrimSpace(payload.Error), strings.TrimSpace(payload.ErrorDescription))
	}
	return nil
}
