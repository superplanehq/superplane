package gitlab

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

type Auth struct {
	client core.HTTPContext
}

func NewAuth(client core.HTTPContext) *Auth {
	return &Auth{client: client}
}

func (a *Auth) RefreshToken(baseURL, clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest(http.MethodPost, baseURL+"/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (a *Auth) exchangeCode(baseURL, clientID, clientSecret, code, redirectURI string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest(http.MethodPost, baseURL+"/oauth/token", strings.NewReader(data.Encode()))
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (a *Auth) HandleCallback(req *http.Request, config *Configuration, expectedState, redirectURI string) (*TokenResponse, error) {
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

	baseURL := config.BaseURL
	
	return a.exchangeCode(baseURL, config.ClientID, config.ClientSecret, code, redirectURI)
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
		return time.Duration(t.ExpiresIn/2) * time.Second
	}
	return time.Hour
}
