package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// AuthorizeURL is where the user approves the OAuth application.
	AuthorizeURL = "https://auth.atlassian.com/authorize"

	// TokenURL exchanges authorization codes and refresh tokens for access tokens.
	TokenURL = "https://auth.atlassian.com/oauth/token"

	// AccessibleResourcesURL lists the Jira Cloud sites the OAuth grant has access to.
	AccessibleResourcesURL = "https://api.atlassian.com/oauth/token/accessible-resources"

	// APIProxyHost is how OAuth apps reach Jira's REST APIs - the site's own domain rejects OAuth bearer tokens.
	APIProxyHost = "https://api.atlassian.com/ex/jira"

	// scopeList is requested on the authorize URL. This integration grants one fixed set of permissions up front.
	// offline_access isn't selectable in the Developer Console's Permissions tab, but it's still required to get a
	// refresh token, so it's requested here directly rather than shown in the setup instructions.
	scopeList = "read:jira-work write:jira-work manage:jira-webhook read:jira-user " +
		"read:servicedesk-request write:servicedesk-request " +
		"read:incident:jira-service-management write:incident:jira-service-management " +
		"read:ops-alert:jira-service-management write:ops-alert:jira-service-management delete:ops-alert:jira-service-management " +
		"read:ops-config:jira-service-management write:ops-config:jira-service-management delete:ops-config:jira-service-management " +
		"offline_access"
)

type Auth struct {
	client core.HTTPContext
}

func NewAuth(client core.HTTPContext) *Auth {
	return &Auth{client: client}
}

// TokenResponse is Atlassian's OAuth token endpoint response. Refresh tokens
// rotate on every use, so both fields are replaced together on refresh.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// GetExpiration returns how long to wait before resyncing to refresh the
// token: half its lifetime, so one failed resync still leaves time to retry.
func (t *TokenResponse) GetExpiration() time.Duration {
	if t.ExpiresIn > 0 {
		seconds := max(t.ExpiresIn/2, 1)
		return time.Duration(seconds) * time.Second
	}
	return 30 * time.Minute
}

func (a *Auth) ExchangeCode(clientID, clientSecret, code, redirectURI string) (*TokenResponse, error) {
	return a.requestToken(map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          code,
		"redirect_uri":  redirectURI,
	})
}

func (a *Auth) RefreshToken(clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	return a.requestToken(map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"refresh_token": refreshToken,
	})
}

func (a *Auth) requestToken(body map[string]string) (*TokenResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error marshaling token request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, TokenURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("error building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error exchanging token: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading token response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("token request got %d: %s", res.StatusCode, string(responseBody))
	}

	var token TokenResponse
	if err := json.Unmarshal(responseBody, &token); err != nil {
		return nil, fmt.Errorf("error parsing token response: %w", err)
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}

	return &token, nil
}

// HandleCallback validates the OAuth callback request and exchanges its code for tokens.
func (a *Auth) HandleCallback(req *http.Request, clientID, clientSecret, expectedState, redirectURI string) (*TokenResponse, error) {
	query := req.URL.Query()
	code := query.Get("code")
	state := query.Get("state")

	if errParam := query.Get("error"); errParam != "" {
		return nil, fmt.Errorf("OAuth error: %s - %s", errParam, query.Get("error_description"))
	}
	if code == "" || state == "" {
		return nil, fmt.Errorf("missing code or state")
	}
	if expectedState == "" || state != expectedState {
		return nil, fmt.Errorf("invalid state")
	}

	return a.ExchangeCode(clientID, clientSecret, code, redirectURI)
}

// AccessibleResource is one Jira Cloud site the OAuth grant has access to.
type AccessibleResource struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Scopes []string `json:"scopes"`
}

func (a *Auth) AccessibleResources(accessToken string) ([]AccessibleResource, error) {
	req, err := http.NewRequest(http.MethodGet, AccessibleResourcesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error building accessible-resources request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	res, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching accessible resources: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading accessible resources response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("accessible resources request got %d: %s", res.StatusCode, string(body))
	}

	var resources []AccessibleResource
	if err := json.Unmarshal(body, &resources); err != nil {
		return nil, fmt.Errorf("error parsing accessible resources response: %w", err)
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("no accessible Jira sites were granted to this OAuth app")
	}

	return resources, nil
}
