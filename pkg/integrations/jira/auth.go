package jira

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	AuthorizationURL = "https://auth.atlassian.com/authorize"
	TokenURL         = "https://auth.atlassian.com/oauth/token"
	APIBaseURL       = "https://api.atlassian.com"
	Audience         = "api.atlassian.com"

	// Secret keys held on the integration.
	OAuthAccessToken  = "accessToken"
	OAuthRefreshToken = "refreshToken"
)

// Scopes requested when initiating the OAuth flow. Classic scopes are used so
// the OAuth app does not need granular permissions enabled.
var oauthScopes = []string{
	"read:jira-user",
	"read:jira-work",
	"write:jira-work",
	"manage:jira-webhook",
	"offline_access",
}

// scopeList returns the scopes joined by space, ready to be put into the
// `scope` query parameter.
func scopeList() string {
	return strings.Join(oauthScopes, " ")
}

type Auth struct {
	client core.HTTPContext
}

func NewAuth(client core.HTTPContext) *Auth {
	return &Auth{client: client}
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// GetExpiration returns half the remaining token TTL so we resync well before
// expiry. Defaults to one hour when expires_in is absent.
func (t *TokenResponse) GetExpiration() time.Duration {
	if t.ExpiresIn > 0 {
		seconds := t.ExpiresIn / 2
		if seconds < 1 {
			seconds = 1
		}
		return time.Duration(seconds) * time.Second
	}
	return time.Hour
}

func (a *Auth) ExchangeCode(clientID, clientSecret, code, redirectURI string) (*TokenResponse, error) {
	body := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          code,
		"redirect_uri":  redirectURI,
	}
	return a.postToken(body)
}

func (a *Auth) RefreshToken(clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	body := map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"refresh_token": refreshToken,
	}
	return a.postToken(body)
}

func (a *Auth) postToken(payload map[string]string) (*TokenResponse, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, TokenURL, strings.NewReader(string(data)))
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %v", err)
	}

	return &tokenResp, nil
}

func (a *Auth) HandleCallback(req *http.Request, clientID, clientSecret, expectedState, redirectURI string) (*TokenResponse, error) {
	code := req.URL.Query().Get("code")
	state := req.URL.Query().Get("state")

	if errorParam := req.URL.Query().Get("error"); errorParam != "" {
		return nil, fmt.Errorf("OAuth error: %s - %s", errorParam, req.URL.Query().Get("error_description"))
	}

	if code == "" || state == "" {
		return nil, fmt.Errorf("missing code or state in callback")
	}

	if state != expectedState {
		return nil, fmt.Errorf("invalid state")
	}

	return a.ExchangeCode(clientID, clientSecret, code, redirectURI)
}

// AccessibleResource describes a Jira Cloud site the OAuth token can access.
type AccessibleResource struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	AvatarURL string   `json:"avatarUrl"`
}

// ListAccessibleResources returns the Jira Cloud sites the token has access
// to. Used during sync to discover the cloud ID.
func (a *Auth) ListAccessibleResources(accessToken string) ([]AccessibleResource, error) {
	req, err := http.NewRequest(http.MethodGet, APIBaseURL+"/oauth/token/accessible-resources", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("accessible-resources returned %d: %s", resp.StatusCode, string(body))
	}

	var resources []AccessibleResource
	if err := json.Unmarshal(body, &resources); err != nil {
		return nil, fmt.Errorf("failed to parse accessible-resources response: %v", err)
	}

	return resources, nil
}

// BuildAuthorizationURL constructs the URL the user should visit to grant the
// OAuth app access to their Jira site.
func BuildAuthorizationURL(clientID, redirectURI, state string) string {
	values := url.Values{}
	values.Set("audience", Audience)
	values.Set("client_id", clientID)
	values.Set("scope", scopeList())
	values.Set("redirect_uri", redirectURI)
	values.Set("state", state)
	values.Set("response_type", "code")
	values.Set("prompt", "consent")
	return AuthorizationURL + "?" + values.Encode()
}
