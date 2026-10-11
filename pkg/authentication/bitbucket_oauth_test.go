package authentication

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBitbucketProviderExchangesAtlassianAuthorizationCode(t *testing.T) {
	var tokenRequest map[string]string
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&tokenRequest))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"token_type":"Bearer"}`)
	}))
	t.Cleanup(tokenServer.Close)

	profile := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
		if r.URL.Path == "/user" {
			_, _ = io.WriteString(w, `{"uuid":"{11111111-1111-1111-1111-111111111111}","username":"ada-bb","display_name":"Ada"}`)
			return
		}
		_, _ = io.WriteString(w, `{"values":[{"email":"ada@example.com","is_primary":true,"is_confirmed":true}]}`)
	}))
	t.Cleanup(profile.Close)

	provider := &bitbucketProvider{
		clientKey:         "client-id",
		secret:            "client-secret",
		callbackURL:       "http://localhost:8000/auth/bitbucket/callback",
		atlassianTokenURL: tokenServer.URL,
		legacyTokenURL:    "http://127.0.0.1:1/legacy",
		userURL:           profile.URL + "/user",
		emailURL:          profile.URL + "/emails",
	}
	session := &bitbucketSession{}
	_, err := session.Authorize(provider, url.Values{"code": {atlassianCode()}})
	require.NoError(t, err)
	assert.Equal(t, "authorization_code", tokenRequest["grant_type"])
	assert.Equal(t, "client-id", tokenRequest["client_id"])
	assert.Equal(t, "client-secret", tokenRequest["client_secret"])
	assert.Equal(t, provider.callbackURL, tokenRequest["redirect_uri"])

	user, err := provider.FetchUser(session)
	require.NoError(t, err)
	assert.Equal(t, "{11111111-1111-1111-1111-111111111111}", user.UserID)
	assert.Equal(t, "ada-bb", user.NickName)
	assert.Equal(t, "ada@example.com", user.Email)
}

func TestBitbucketProviderKeepsLegacyAuthorizationCodeOnBitbucket(t *testing.T) {
	legacyCalled := false
	legacy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		legacyCalled = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"legacy-token","token_type":"Bearer"}`)
	}))
	t.Cleanup(legacy.Close)

	provider := &bitbucketProvider{
		clientKey:         "client-id",
		secret:            "client-secret",
		callbackURL:       "http://localhost:8000/auth/bitbucket/callback",
		legacyTokenURL:    legacy.URL,
		atlassianTokenURL: "http://127.0.0.1:1/atlassian",
	}
	session := &bitbucketSession{}
	token, err := session.Authorize(provider, url.Values{"code": {"opaque-bitbucket-code"}})
	require.NoError(t, err)
	assert.True(t, legacyCalled)
	assert.Equal(t, "legacy-token", token)
}

func TestBitbucketSessionOmitsTheAccessTokenFromTheCookie(t *testing.T) {
	session := &bitbucketSession{
		AuthURL:      "https://bitbucket.org/site/oauth2/authorize?state=connect",
		AccessToken:  strings.Repeat("t", 3000),
		RefreshToken: strings.Repeat("r", 3000),
	}

	stored := session.Marshal()

	assert.Less(t, len(stored), 500)
	assert.NotContains(t, stored, session.AccessToken)
	assert.Contains(t, stored, "connect")
}

func atlassianCode() string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"auth.atlassian.com","https://id.atlassian.com/atl_token_type":"AUTH_CODE"}`))
	return header + "." + payload + ".signature"
}
