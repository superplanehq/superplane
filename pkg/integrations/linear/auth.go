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
	linearAuthorizeURL = "https://linear.app/oauth/authorize"
	linearTokenURL     = "https://api.linear.app/oauth/token"
)

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

func (a *Auth) exchangeCode(clientID, clientSecret, code, redirectURI string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	return a.postToken(data)
}

func (a *Auth) RefreshToken(clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)
	return a.postToken(data)
}

func (a *Auth) postToken(data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequest(http.MethodPost, linearTokenURL, strings.NewReader(data.Encode()))
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

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (a *Auth) HandleCallback(req *http.Request, clientID, clientSecret, expectedState, redirectURI string) (*TokenResponse, error) {
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

	return a.exchangeCode(clientID, clientSecret, code, redirectURI)
}

func findSecret(integration core.IntegrationContext, name string) (string, error) {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return "", err
	}
	for _, secret := range secrets {
		if secret.Name == name {
			return string(secret.Value), nil
		}
	}
	return "", nil
}
