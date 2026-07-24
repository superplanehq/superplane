package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	atlassianAuthorizeURL = "https://auth.atlassian.com/authorize"
	atlassianTokenURL     = "https://auth.atlassian.com/oauth/token"
	atlassianResourcesURL = "https://api.atlassian.com/oauth/token/accessible-resources"

	// atlassianAPIProxyHost is how OAuth apps reach Jira's REST APIs - the site's own domain rejects OAuth bearer tokens.
	atlassianAPIProxyHost = "https://api.atlassian.com/ex/jira"

	// oauthScopes is the fixed scope set requested for this PoC.
	oauthScopes = "read:jira-work write:jira-work manage:jira-webhook read:jira-user offline_access"
)

// BuildAuthorizeURL returns the Atlassian OAuth 2.0 (3LO) URL the user's browser is redirected to.
func BuildAuthorizeURL(clientID, redirectURI, state string) string {
	query := url.Values{}
	query.Set("audience", "api.atlassian.com")
	query.Set("client_id", clientID)
	query.Set("scope", oauthScopes)
	query.Set("redirect_uri", redirectURI)
	query.Set("state", state)
	query.Set("response_type", "code")
	query.Set("prompt", "consent")
	return atlassianAuthorizeURL + "?" + query.Encode()
}

// oauthToken is the response shape of Atlassian's token endpoint.
type oauthToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
}

func exchangeCodeForToken(httpCtx core.HTTPContext, clientID, clientSecret, code, redirectURI string) (*oauthToken, error) {
	return doTokenRequest(httpCtx, map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          code,
		"redirect_uri":  redirectURI,
	})
}

func refreshAccessToken(httpCtx core.HTTPContext, clientID, clientSecret, refreshToken string) (*oauthToken, error) {
	return doTokenRequest(httpCtx, map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"refresh_token": refreshToken,
	})
}

func doTokenRequest(httpCtx core.HTTPContext, body map[string]string) (*oauthToken, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error marshaling token request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, atlassianTokenURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("error building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := httpCtx.Do(req)
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

	var token oauthToken
	if err := json.Unmarshal(responseBody, &token); err != nil {
		return nil, fmt.Errorf("error parsing token response: %w", err)
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}

	return &token, nil
}

// AccessibleResource is one Jira Cloud site the OAuth grant has access to.
type AccessibleResource struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Scopes []string `json:"scopes"`
}

func fetchAccessibleResources(httpCtx core.HTTPContext, accessToken string) ([]AccessibleResource, error) {
	req, err := http.NewRequest(http.MethodGet, atlassianResourcesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error building accessible-resources request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	res, err := httpCtx.Do(req)
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
