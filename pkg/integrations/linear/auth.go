package linear

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
	// AuthorizeURL is where the user approves the OAuth application.
	AuthorizeURL = "https://linear.app/oauth/authorize"

	// TokenURL exchanges authorization codes and refresh tokens for access tokens.
	TokenURL = "https://api.linear.app/oauth/token"

	// AppsNewURL is Linear's OAuth application creation form. There is no API to
	// create OAuth apps, but the form accepts manifest query parameters that
	// pre-fill every field, so the user only has to click Create.
	AppsNewURL = "https://linear.app/settings/api/applications/new"
)

type Auth struct {
	client core.HTTPContext
}

func NewAuth(client core.HTTPContext) *Auth {
	return &Auth{client: client}
}

// TokenResponse is Linear's token endpoint response. Since April 2026 Linear
// issues 24-hour access tokens with rotating refresh tokens, so both fields
// must be stored and refresh handled on resync.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// GetExpiration returns how long to wait before resyncing to refresh the
// token: half its lifetime, so one failed resync still leaves time to retry.
func (t *TokenResponse) GetExpiration() time.Duration {
	if t.ExpiresIn > 0 {
		seconds := max(t.ExpiresIn/2, 1)
		return time.Duration(seconds) * time.Second
	}
	return time.Hour
}

// ExpiresAt returns when the access token stops working, or the zero time when
// Linear did not report a lifetime.
func (t *TokenResponse) ExpiresAt() time.Time {
	if t.ExpiresIn <= 0 {
		return time.Time{}
	}
	return time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)
}

func (a *Auth) ExchangeCode(clientID, clientSecret, code, redirectURI string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	return a.requestToken(data)
}

func (a *Auth) RefreshToken(clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)

	return a.requestToken(data)
}

func (a *Auth) requestToken(data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequest(http.MethodPost, TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	tokenResponse := TokenResponse{}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, err
	}

	if tokenResponse.AccessToken == "" {
		return nil, fmt.Errorf("token response contained no access token")
	}

	return &tokenResponse, nil
}

func (a *Auth) HandleCallback(req *http.Request, clientID, clientSecret, expectedState, redirectURI string) (*TokenResponse, error) {
	code := req.URL.Query().Get("code")
	state := req.URL.Query().Get("state")
	errorParam := req.URL.Query().Get("error")

	if errorParam != "" {
		errorDescription := req.URL.Query().Get("error_description")
		return nil, fmt.Errorf("OAuth error: %s - %s", errorParam, errorDescription)
	}

	if code == "" || state == "" {
		return nil, fmt.Errorf("missing code or state")
	}

	if expectedState == "" || state != expectedState {
		return nil, fmt.Errorf("invalid state")
	}

	return a.ExchangeCode(clientID, clientSecret, code, redirectURI)
}
