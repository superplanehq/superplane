package authentication

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/markbates/goth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestBitbucketConnectProvider_RequestsAccountScope(t *testing.T) {
	provider := newBitbucketConnectProvider(ProviderConfig{
		Key:         "consumer-key",
		Secret:      "consumer-secret",
		CallbackURL: "https://app.example/auth/bitbucket/callback",
	})

	session, err := provider.BeginAuth("connect-state")
	require.NoError(t, err)
	authURL, err := session.GetAuthURL()
	require.NoError(t, err)

	parsed, err := url.Parse(authURL)
	require.NoError(t, err)
	assert.Equal(t, "bitbucket.org", parsed.Host)
	assert.Equal(t, "/site/oauth2/authorize", parsed.Path)
	assert.Equal(t, bitbucketAccountScope, parsed.Query().Get("scope"))
	assert.Equal(t, models.ProviderBitbucket, provider.Name())
}

func TestBitbucketConnectProvider_FetchUser_MapsAccountWithoutEmail(t *testing.T) {
	provider := newBitbucketConnectProvider(ProviderConfig{
		Key:         "consumer-key",
		Secret:      "consumer-secret",
		CallbackURL: "https://app.example/auth/bitbucket/callback",
	})
	connect := provider.(*bitbucketConnectProvider)
	var requested []string
	connect.inner.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requested = append(requested, request.URL.String())
		switch request.URL.String() {
		case "https://bitbucket.org/site/oauth2/access_token":
			assert.Equal(t, http.MethodPost, request.Method)
			return jsonResponse(`{"access_token":"account-token","token_type":"bearer","refresh_token":"refresh-token"}`), nil
		case bitbucketAccountEndpoint:
			assert.Equal(t, http.MethodGet, request.Method)
			assert.Equal(t, "Bearer account-token", request.Header.Get("Authorization"))
			return jsonResponse(`{
				"uuid": "{504C3B1A-1001-4000-8000-000000000001}",
				"username": "removed-field",
				"nickname": "ada",
				"display_name": "Ada Lovelace",
				"links": {"avatar": {"href": "https://bitbucket.org/account/ada/avatar/32/"}}
			}`), nil
		default:
			return jsonResponse(`{"type":"error"}`), &unexpectedBitbucketRequest{url: request.URL.String()}
		}
	})}

	started, err := provider.BeginAuth("state")
	require.NoError(t, err)
	session, err := provider.UnmarshalSession(started.Marshal())
	require.NoError(t, err)

	_, err = provider.FetchUser(session)
	require.Error(t, err)

	_, err = session.Authorize(provider, url.Values{"code": {"abc"}})
	require.NoError(t, err)

	user, err := provider.FetchUser(session)
	require.NoError(t, err)
	assert.Equal(t, "504c3b1a-1001-4000-8000-000000000001", user.UserID)
	assert.Equal(t, "ada", user.NickName)
	assert.Equal(t, "Ada Lovelace", user.Name)
	assert.Equal(t, "https://bitbucket.org/account/ada/avatar/32/", user.AvatarURL)
	assert.Equal(t, models.ProviderBitbucket, user.Provider)
	assert.Empty(t, user.Email)
	assert.Empty(t, user.AccessToken)
	assert.Empty(t, user.RefreshToken)
	assert.NotContains(t, requested, "https://api.bitbucket.org/2.0/user/emails")
}

func TestBitbucketAccountFromBody_RejectsInvalidIdentity(t *testing.T) {
	_, err := bitbucketAccountFromBody([]byte(`{"uuid":"not-a-uuid","nickname":"ada"}`))
	require.Error(t, err)

	_, err = bitbucketAccountFromBody([]byte(`{"uuid":"{504c3b1a-1001-4000-8000-000000000001}","nickname":"  ","username":"removed-field"}`))
	require.Error(t, err)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type unexpectedBitbucketRequest struct {
	url string
}

func (e *unexpectedBitbucketRequest) Error() string {
	return "unexpected bitbucket request " + e.url
}

var _ goth.Provider = (*bitbucketConnectProvider)(nil)
