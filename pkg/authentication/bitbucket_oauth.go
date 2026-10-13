package authentication

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/markbates/goth"
	"golang.org/x/oauth2"
)

const (
	bitbucketAuthURL           = "https://bitbucket.org/site/oauth2/authorize"
	bitbucketLegacyTokenURL    = "https://bitbucket.org/site/oauth2/access_token"
	bitbucketAtlassianTokenURL = "https://auth.atlassian.com/oauth/token"
	bitbucketUserURL           = "https://api.bitbucket.org/2.0/user"
	bitbucketEmailURL          = "https://api.bitbucket.org/2.0/user/emails"
	atlassianIssuer            = "auth.atlassian.com"
)

// bitbucketProvider links a Bitbucket account. New Bitbucket OAuth clients
// return an Atlassian authorization code. Older consumers still return a
// Bitbucket code. The profile call is the same for both.
type bitbucketProvider struct {
	clientKey         string
	secret            string
	callbackURL       string
	authURL           string
	legacyTokenURL    string
	atlassianTokenURL string
	userURL           string
	emailURL          string
	httpClient        *http.Client
}

func newBitbucketProvider(clientKey, secret, callbackURL string) goth.Provider {
	return &bitbucketProvider{
		clientKey:         clientKey,
		secret:            secret,
		callbackURL:       callbackURL,
		authURL:           bitbucketAuthURL,
		legacyTokenURL:    bitbucketLegacyTokenURL,
		atlassianTokenURL: bitbucketAtlassianTokenURL,
		userURL:           bitbucketUserURL,
		emailURL:          bitbucketEmailURL,
	}
}

func (p *bitbucketProvider) Name() string {
	return "bitbucket"
}

func (p *bitbucketProvider) SetName(string) {}

func (p *bitbucketProvider) Debug(bool) {}

func (p *bitbucketProvider) BeginAuth(state string) (goth.Session, error) {
	return &bitbucketSession{AuthURL: p.oauthConfig(p.legacyTokenURL).AuthCodeURL(state)}, nil
}

func (p *bitbucketProvider) UnmarshalSession(data string) (goth.Session, error) {
	session := &bitbucketSession{}
	err := json.Unmarshal([]byte(data), session)
	return session, err
}

func (p *bitbucketProvider) FetchUser(session goth.Session) (goth.User, error) {
	bitbucketSession, ok := session.(*bitbucketSession)
	if !ok {
		return goth.User{}, fmt.Errorf("bitbucket session is invalid")
	}
	user := goth.User{
		AccessToken:  bitbucketSession.AccessToken,
		Provider:     p.Name(),
		RefreshToken: bitbucketSession.RefreshToken,
		ExpiresAt:    bitbucketSession.ExpiresAt,
	}
	if user.AccessToken == "" {
		return user, fmt.Errorf("bitbucket cannot get user information without an access token")
	}
	if err := p.readUser(&user); err != nil {
		return user, err
	}
	p.readEmail(&user)
	return user, nil
}

func (p *bitbucketProvider) RefreshToken(refreshToken string) (*oauth2.Token, error) {
	tokenSource := p.oauthConfig(p.legacyTokenURL).TokenSource(context.Background(), &oauth2.Token{RefreshToken: refreshToken})
	return tokenSource.Token()
}

func (p *bitbucketProvider) RefreshTokenAvailable() bool {
	return true
}

func (p *bitbucketProvider) client() *http.Client {
	if p.httpClient != nil {
		return p.httpClient
	}
	return http.DefaultClient
}

func (p *bitbucketProvider) oauthConfig(tokenURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.clientKey,
		ClientSecret: p.secret,
		RedirectURL:  p.callbackURL,
		Scopes:       []string{"account"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  p.authURL,
			TokenURL: tokenURL,
		},
	}
}

func (p *bitbucketProvider) exchange(code string) (*oauth2.Token, error) {
	if atlassianAuthorizationCode(code) {
		return p.exchangeAtlassian(code)
	}
	return p.oauthConfig(p.legacyTokenURL).Exchange(context.Background(), code)
}

func (p *bitbucketProvider) exchangeAtlassian(code string) (*oauth2.Token, error) {
	payload, err := json.Marshal(map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     p.clientKey,
		"client_secret": p.secret,
		"code":          code,
		"redirect_uri":  p.callbackURL,
	})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest(http.MethodPost, p.atlassianTokenURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := p.client().Do(request)
	if err != nil {
		return nil, fmt.Errorf("exchange bitbucket authorization code: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &failure)
		if failure.Error != "" {
			return nil, fmt.Errorf("bitbucket authorization code was rejected (%d): %s", response.StatusCode, failure.Error)
		}
		return nil, fmt.Errorf("bitbucket authorization code was rejected (%d)", response.StatusCode)
	}
	var tokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("read bitbucket access token: %w", err)
	}
	if tokenResponse.AccessToken == "" {
		return nil, fmt.Errorf("bitbucket returned no access token")
	}
	token := &oauth2.Token{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
		TokenType:    tokenResponse.TokenType,
	}
	if tokenResponse.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
	}
	return token, nil
}

func (p *bitbucketProvider) readUser(user *goth.User) error {
	var profile struct {
		UUID        string `json:"uuid"`
		Username    string `json:"username"`
		Nickname    string `json:"nickname"`
		DisplayName string `json:"display_name"`
		Links       struct {
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"links"`
	}
	if err := p.getJSON(p.userURL, user.AccessToken, &profile); err != nil {
		return err
	}
	user.UserID = profile.UUID
	user.NickName = strings.TrimSpace(profile.Username)
	if user.NickName == "" {
		user.NickName = strings.TrimSpace(profile.Nickname)
	}
	user.Name = profile.DisplayName
	user.AvatarURL = profile.Links.Avatar.Href
	return nil
}

func (p *bitbucketProvider) readEmail(user *goth.User) {
	var emails struct {
		Values []struct {
			Email       string `json:"email"`
			IsPrimary   bool   `json:"is_primary"`
			IsConfirmed bool   `json:"is_confirmed"`
		} `json:"values"`
	}
	if err := p.getJSON(p.emailURL, user.AccessToken, &emails); err != nil {
		return
	}
	for _, email := range emails.Values {
		if email.IsPrimary && email.IsConfirmed {
			user.Email = email.Email
			return
		}
	}
}

func (p *bitbucketProvider) getJSON(rawURL, accessToken string, dest any) error {
	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := p.client().Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("bitbucket profile returned %d", response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(dest)
}

func atlassianAuthorizationCode(code string) bool {
	parts := strings.Split(code, ".")
	if len(parts) != 3 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims struct {
		Issuer    string `json:"iss"`
		TokenType string `json:"https://id.atlassian.com/atl_token_type"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return false
	}
	return claims.Issuer == atlassianIssuer || claims.TokenType == "AUTH_CODE"
}

type bitbucketSession struct {
	AuthURL      string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func (s *bitbucketSession) GetAuthURL() (string, error) {
	if s.AuthURL == "" {
		return "", fmt.Errorf("bitbucket auth URL is empty")
	}
	return s.AuthURL, nil
}

func (s *bitbucketSession) Authorize(provider goth.Provider, params goth.Params) (string, error) {
	bitbucket, ok := provider.(*bitbucketProvider)
	if !ok {
		return "", fmt.Errorf("bitbucket provider is invalid")
	}
	token, err := bitbucket.exchange(params.Get("code"))
	if err != nil {
		return "", err
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("bitbucket returned no access token")
	}
	s.AccessToken = token.AccessToken
	s.RefreshToken = token.RefreshToken
	s.ExpiresAt = token.Expiry
	return token.AccessToken, nil
}

// Marshal stores only the authorization URL. The access token stays in memory
// for the profile request. Writing it into the cookie exceeds the 4KB cookie
// limit because Atlassian access tokens are large.
func (s *bitbucketSession) Marshal() string {
	payload, _ := json.Marshal(bitbucketSession{AuthURL: s.AuthURL})
	return string(payload)
}

func (s *bitbucketSession) String() string {
	return s.Marshal()
}
