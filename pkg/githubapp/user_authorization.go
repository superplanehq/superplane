package githubapp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	userAuthorizeURL = "https://github.com/login/oauth/authorize"
	userTokenURL     = "https://github.com/login/oauth/access_token"
	githubAPIURL     = "https://api.github.com"
)

var (
	ErrUserAuthorizationUnavailable = errors.New("GitHub App user authorization is not configured")
	ErrBadRefreshToken              = errors.New("GitHub refresh token is no longer valid")
)

// UserTokens is a GitHub App user access token pair. Expiry values are nil
// when the app does not expire user tokens.
type UserTokens struct {
	AccessToken           string
	AccessTokenExpiresAt  *time.Time
	RefreshToken          string
	RefreshTokenExpiresAt *time.Time
}

type GitHubUser struct {
	ID        int64
	Login     string
	Name      string
	AvatarURL string
}

type userTokenResponse struct {
	AccessToken           string `json:"access_token"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
	Error                 string `json:"error"`
	ErrorDescription      string `json:"error_description"`
}

// NewPKCEVerifier returns a random PKCE code verifier.
func NewPKCEVerifier() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate PKCE verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func PKCEChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// UserAuthorizationURL is the GitHub page where a member authorizes the app.
func UserAuthorizationURL(cfg config.GitHubHostedAppConfig, redirectURI, state, verifier string) (string, error) {
	if !cfg.UserAuthorizationEnabled() {
		return "", ErrUserAuthorizationUnavailable
	}
	query := url.Values{}
	query.Set("client_id", cfg.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("state", state)
	query.Set("code_challenge", PKCEChallenge(verifier))
	query.Set("code_challenge_method", "S256")
	query.Set("prompt", "select_account")
	return userAuthorizeURL + "?" + query.Encode(), nil
}

func ExchangeUserCode(
	httpCtx core.HTTPContext,
	cfg config.GitHubHostedAppConfig,
	code, verifier, redirectURI string,
) (*UserTokens, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("code_verifier", verifier)
	form.Set("redirect_uri", redirectURI)
	return requestUserTokens(httpCtx, cfg, form)
}

// RefreshUserTokens rotates the token pair. GitHub revokes the refresh token
// that this call uses.
func RefreshUserTokens(httpCtx core.HTTPContext, cfg config.GitHubHostedAppConfig, refreshToken string) (*UserTokens, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	return requestUserTokens(httpCtx, cfg, form)
}

func FetchGitHubUser(httpCtx core.HTTPContext, accessToken string) (*GitHubUser, error) {
	request, err := http.NewRequest(http.MethodGet, githubAPIURL+"/user", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+accessToken)

	body, status, err := doRequest(httpCtx, request)
	if err != nil {
		return nil, fmt.Errorf("fetch GitHub user: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("fetch GitHub user: GitHub returned %d", status)
	}

	var user struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("decode GitHub user: %w", err)
	}
	if user.ID <= 0 || strings.TrimSpace(user.Login) == "" {
		return nil, errors.New("GitHub user response is incomplete")
	}
	return &GitHubUser{ID: user.ID, Login: user.Login, Name: user.Name, AvatarURL: user.AvatarURL}, nil
}

// RevokeUserToken removes the token at GitHub. GitHub answers 404 when the
// token is already gone.
func RevokeUserToken(httpCtx core.HTTPContext, cfg config.GitHubHostedAppConfig, accessToken string) error {
	if !cfg.UserAuthorizationEnabled() {
		return ErrUserAuthorizationUnavailable
	}
	payload, err := json.Marshal(map[string]string{"access_token": accessToken})
	if err != nil {
		return err
	}
	request, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/applications/%s/token", githubAPIURL, url.PathEscape(cfg.ClientID)),
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return err
	}
	request.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Content-Type", "application/json")

	_, status, err := doRequest(httpCtx, request)
	if err != nil {
		return fmt.Errorf("revoke GitHub user token: %w", err)
	}
	if status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("revoke GitHub user token: GitHub returned %d", status)
	}
	return nil
}

func requestUserTokens(httpCtx core.HTTPContext, cfg config.GitHubHostedAppConfig, form url.Values) (*UserTokens, error) {
	if !cfg.UserAuthorizationEnabled() {
		return nil, ErrUserAuthorizationUnavailable
	}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)

	request, err := http.NewRequest(http.MethodPost, userTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	body, status, err := doRequest(httpCtx, request)
	if err != nil {
		return nil, fmt.Errorf("request GitHub user token: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("request GitHub user token: GitHub returned %d", status)
	}

	var response userTokenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode GitHub user token: %w", err)
	}
	if response.Error == "bad_refresh_token" {
		return nil, ErrBadRefreshToken
	}
	if response.Error != "" {
		return nil, fmt.Errorf("request GitHub user token: %s", response.Error)
	}
	if response.AccessToken == "" {
		return nil, errors.New("GitHub user token response is incomplete")
	}

	now := time.Now()
	return &UserTokens{
		AccessToken:           response.AccessToken,
		AccessTokenExpiresAt:  expiresAt(now, response.ExpiresIn),
		RefreshToken:          response.RefreshToken,
		RefreshTokenExpiresAt: expiresAt(now, response.RefreshTokenExpiresIn),
	}, nil
}

func doRequest(httpCtx core.HTTPContext, request *http.Request) ([]byte, int, error) {
	if httpCtx == nil {
		return nil, 0, errors.New("HTTP client is required")
	}
	response, err := httpCtx.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, 0, err
	}
	return body, response.StatusCode, nil
}

func expiresAt(now time.Time, seconds int64) *time.Time {
	if seconds <= 0 {
		return nil
	}
	value := now.Add(time.Duration(seconds) * time.Second)
	return &value
}

// UserIDString formats the GitHub user id the way linked accounts store it.
func (u *GitHubUser) UserIDString() string {
	return strconv.FormatInt(u.ID, 10)
}
