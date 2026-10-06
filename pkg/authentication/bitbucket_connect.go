package authentication

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/bitbucket"
	"golang.org/x/oauth2"
)

const bitbucketAccountEndpoint = "https://api.bitbucket.org/2.0/user"

const bitbucketAccountScope = "account"

type bitbucketConnectProvider struct {
	inner *bitbucket.Provider
}

func newBitbucketConnectProvider(config ProviderConfig) goth.Provider {
	return &bitbucketConnectProvider{
		inner: bitbucket.New(config.Key, config.Secret, config.CallbackURL, bitbucketAccountScope),
	}
}

func (p *bitbucketConnectProvider) Name() string {
	return p.inner.Name()
}

func (p *bitbucketConnectProvider) SetName(name string) {
	p.inner.SetName(name)
}

func (p *bitbucketConnectProvider) Debug(debug bool) {
	p.inner.Debug(debug)
}

func (p *bitbucketConnectProvider) BeginAuth(state string) (goth.Session, error) {
	session, err := p.inner.BeginAuth(state)
	if err != nil {
		return nil, err
	}
	return &bitbucketConnectSession{inner: session, provider: p.inner}, nil
}

func (p *bitbucketConnectProvider) UnmarshalSession(data string) (goth.Session, error) {
	session, err := p.inner.UnmarshalSession(data)
	if err != nil {
		return nil, err
	}
	return &bitbucketConnectSession{inner: session, provider: p.inner}, nil
}

func (p *bitbucketConnectProvider) FetchUser(session goth.Session) (goth.User, error) {
	connectSession, ok := session.(*bitbucketConnectSession)
	if !ok {
		return goth.User{}, fmt.Errorf("invalid bitbucket session")
	}
	bitbucketSession, ok := connectSession.inner.(*bitbucket.Session)
	if !ok {
		return goth.User{}, fmt.Errorf("invalid bitbucket session")
	}
	if bitbucketSession.AccessToken == "" {
		return goth.User{}, fmt.Errorf("bitbucket cannot get user information without accessToken")
	}

	request, err := http.NewRequest(http.MethodGet, bitbucketAccountEndpoint, nil)
	if err != nil {
		return goth.User{}, err
	}
	request.Header.Set("Authorization", "Bearer "+bitbucketSession.AccessToken)

	response, err := p.inner.Client().Do(request)
	if err != nil {
		return goth.User{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return goth.User{}, fmt.Errorf("bitbucket responded with %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return goth.User{}, err
	}

	user, err := bitbucketAccountFromBody(body)
	if err != nil {
		return goth.User{}, err
	}
	user.Provider = p.Name()
	return user, nil
}

func (p *bitbucketConnectProvider) RefreshToken(refreshToken string) (*oauth2.Token, error) {
	return p.inner.RefreshToken(refreshToken)
}

func (p *bitbucketConnectProvider) RefreshTokenAvailable() bool {
	return p.inner.RefreshTokenAvailable()
}

type bitbucketConnectSession struct {
	inner    goth.Session
	provider *bitbucket.Provider
}

func (s *bitbucketConnectSession) GetAuthURL() (string, error) {
	return s.inner.GetAuthURL()
}

func (s *bitbucketConnectSession) Marshal() string {
	return s.inner.Marshal()
}

func (s *bitbucketConnectSession) Authorize(_ goth.Provider, params goth.Params) (string, error) {
	return s.inner.Authorize(s.provider, params)
}

func bitbucketAccountFromBody(body []byte) (goth.User, error) {
	var account struct {
		UUID        string `json:"uuid"`
		Nickname    string `json:"nickname"`
		DisplayName string `json:"display_name"`
		Links       struct {
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"links"`
	}
	if err := json.Unmarshal(body, &account); err != nil {
		return goth.User{}, err
	}

	parsed, err := uuid.Parse(account.UUID)
	if err != nil {
		return goth.User{}, fmt.Errorf("bitbucket returned an invalid account UUID")
	}
	nickname := strings.TrimSpace(account.Nickname)
	if nickname == "" {
		return goth.User{}, fmt.Errorf("bitbucket returned no nickname")
	}

	return goth.User{
		UserID:    parsed.String(),
		NickName:  nickname,
		Name:      account.DisplayName,
		AvatarURL: account.Links.Avatar.Href,
	}, nil
}
