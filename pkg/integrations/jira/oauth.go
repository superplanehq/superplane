package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	AuthTypeAPIToken  = "apiToken"
	AuthTypeOAuth     = "oauth"
	OAuthAccessToken  = "accessToken"
	OAuthRefreshToken = "refreshToken"

	atlassianAuthURL      = "https://auth.atlassian.com/authorize"
	atlassianTokenURL     = "https://auth.atlassian.com/oauth/token"
	atlassianResourcesURL = "https://api.atlassian.com/oauth/token/accessible-resources"

	// OAuth scopes required for Jira operations and webhook management
	oauthScopes = "read:jira-work write:jira-work read:jira-user manage:jira-webhook offline_access"
)

// OAuthTokenResponse represents the response from Atlassian OAuth token endpoint.
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

// GetExpiration returns the duration until the token expires.
// Returns half the expiration time to allow for early refresh.
func (r *OAuthTokenResponse) GetExpiration() time.Duration {
	if r.ExpiresIn > 0 {
		return time.Duration(r.ExpiresIn/2) * time.Second
	}
	return time.Hour
}

// AccessibleResource represents a Jira Cloud instance the user has access to.
type AccessibleResource struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	AvatarURL string   `json:"avatarUrl"`
}

// buildAuthorizationURL generates the Atlassian authorization URL for OAuth 2.0 (3LO).
func buildAuthorizationURL(clientID, redirectURI, state string) string {
	params := url.Values{}
	params.Set("audience", "api.atlassian.com")
	params.Set("client_id", clientID)
	params.Set("scope", oauthScopes)
	params.Set("redirect_uri", redirectURI)
	params.Set("state", state)
	params.Set("response_type", "code")
	params.Set("prompt", "consent")

	return fmt.Sprintf("%s?%s", atlassianAuthURL, params.Encode())
}

// exchangeCodeForTokens exchanges an authorization code for access and refresh tokens.
func exchangeCodeForTokens(httpCtx core.HTTPContext, clientID, clientSecret, code, redirectURI string) (*OAuthTokenResponse, error) {
	requestBody := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          code,
		"redirect_uri":  redirectURI,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, atlassianTokenURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpCtx.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResponse OAuthTokenResponse
	if err := json.Unmarshal(respBody, &tokenResponse); err != nil {
		return nil, fmt.Errorf("error parsing token response: %v", err)
	}

	return &tokenResponse, nil
}

// refreshOAuthToken refreshes the access token using a refresh token.
func refreshOAuthToken(httpCtx core.HTTPContext, clientID, clientSecret, refreshToken string) (*OAuthTokenResponse, error) {
	requestBody := map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"refresh_token": refreshToken,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, atlassianTokenURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpCtx.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResponse OAuthTokenResponse
	if err := json.Unmarshal(respBody, &tokenResponse); err != nil {
		return nil, fmt.Errorf("error parsing token response: %v", err)
	}

	return &tokenResponse, nil
}

// getAccessibleResources fetches the Jira Cloud instances the user has access to.
func getAccessibleResources(httpCtx core.HTTPContext, accessToken string) ([]AccessibleResource, error) {
	req, err := http.NewRequest(http.MethodGet, atlassianResourcesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := httpCtx.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get accessible resources with status %d: %s", resp.StatusCode, string(respBody))
	}

	var resources []AccessibleResource
	if err := json.Unmarshal(respBody, &resources); err != nil {
		return nil, fmt.Errorf("error parsing resources response: %v", err)
	}

	return resources, nil
}

// findOAuthSecret finds a secret by name from the integration secrets.
func findOAuthSecret(ctx core.IntegrationContext, name string) (string, error) {
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return "", fmt.Errorf("error getting secrets: %v", err)
	}

	for _, secret := range secrets {
		if secret.Name == name {
			return string(secret.Value), nil
		}
	}

	return "", nil
}
