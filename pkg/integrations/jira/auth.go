package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	atlassianAuthorizeURL           = "https://auth.atlassian.com/authorize"
	atlassianTokenURL               = "https://auth.atlassian.com/oauth/token"
	atlassianAccessibleResourcesURL = "https://api.atlassian.com/oauth/token/accessible-resources"
)

type Auth struct {
	client core.HTTPContext
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type AccessibleResource struct {
	ID     string   `json:"id"`
	URL    string   `json:"url"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

func NewAuth(client core.HTTPContext) *Auth {
	return &Auth{client: client}
}

func jiraOAuthURL(clientID, redirectURI, state string) string {
	values := url.Values{}
	values.Set("audience", "api.atlassian.com")
	values.Set("client_id", clientID)
	values.Set("scope", strings.Join(oauthScopes, " "))
	values.Set("redirect_uri", redirectURI)
	values.Set("state", state)
	values.Set("response_type", "code")
	values.Set("prompt", "consent")

	return fmt.Sprintf("%s?%s", atlassianAuthorizeURL, values.Encode())
}

func (a *Auth) HandleCallback(req *http.Request, config Configuration, expectedState, redirectURI string) (*TokenResponse, error) {
	code := req.URL.Query().Get("code")
	state := req.URL.Query().Get("state")
	errorParam := req.URL.Query().Get("error")

	if errorParam != "" {
		errorDesc := req.URL.Query().Get("error_description")
		return nil, fmt.Errorf("OAuth error: %s - %s", errorParam, errorDesc)
	}

	if code == "" || state == "" {
		return nil, fmt.Errorf("missing code or state")
	}

	if state != expectedState {
		return nil, fmt.Errorf("invalid state")
	}

	return a.ExchangeCode(config.ClientID, config.ClientSecret, code, redirectURI)
}

func (a *Auth) ExchangeCode(clientID, clientSecret, code, redirectURI string) (*TokenResponse, error) {
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          code,
		"redirect_uri":  redirectURI,
	}

	return a.tokenRequest(payload)
}

func (a *Auth) RefreshToken(clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	payload := map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"refresh_token": refreshToken,
	}

	return a.tokenRequest(payload)
}

func (a *Auth) tokenRequest(payload map[string]string) (*TokenResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, atlassianTokenURL, bytes.NewReader(body))
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

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed: status %d, body: %s", resp.StatusCode, string(responseBody))
	}

	tokenResponse := TokenResponse{}
	if err := json.Unmarshal(responseBody, &tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}

func (a *Auth) AccessibleResources(accessToken string) ([]AccessibleResource, error) {
	req, err := http.NewRequest(http.MethodGet, atlassianAccessibleResourcesURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("accessible resources request failed: status %d, body: %s", resp.StatusCode, string(responseBody))
	}

	resources := []AccessibleResource{}
	if err := json.Unmarshal(responseBody, &resources); err != nil {
		return nil, err
	}

	return resources, nil
}

func firstJiraResource(resources []AccessibleResource) (*AccessibleResource, error) {
	for i := range resources {
		if slices.Contains(resources[i].Scopes, "read:jira-work") || strings.Contains(resources[i].URL, ".atlassian.net") {
			return &resources[i], nil
		}
	}

	if len(resources) > 0 {
		return &resources[0], nil
	}

	return nil, fmt.Errorf("no Jira sites are available to this OAuth grant")
}
