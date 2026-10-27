package githubapp

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

type queuedHTTP struct {
	responses []*http.Response
	requests  []*http.Request
	bodies    []string
}

func (q *queuedHTTP) Do(request *http.Request) (*http.Response, error) {
	q.requests = append(q.requests, request)
	body := ""
	if request.Body != nil {
		data, _ := io.ReadAll(request.Body)
		body = string(data)
	}
	q.bodies = append(q.bodies, body)
	response := q.responses[0]
	q.responses = q.responses[1:]
	return response, nil
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func userAuthorizationConfig() config.GitHubHostedAppConfig {
	return config.GitHubHostedAppConfig{
		ID:            44,
		Slug:          "superplane",
		PrivateKey:    "pem",
		WebhookSecret: "whsec",
		ClientID:      "Iv1.client",
		ClientSecret:  "client-secret",
	}
}

func createAuthorizedAccount(t *testing.T, tokens *UserTokens) uuid.UUID {
	t.Helper()
	account, err := models.CreateAccount("Octo Cat", "octo-"+uuid.NewString()+"@example.com")
	require.NoError(t, err)
	require.NoError(t, SaveUserAuthorization(
		t.Context(),
		database.DB(t.Context()),
		testEncryptor(),
		account.ID,
		userAuthorizationConfig().ID,
		&GitHubUser{ID: 583231, Login: "octocat"},
		tokens,
	))
	return account.ID
}

func timeIn(duration time.Duration) *time.Time {
	value := time.Now().Add(duration)
	return &value
}

func TestUserAuthorizationURLUsesPKCE(t *testing.T) {
	authorizeURL, err := UserAuthorizationURL(userAuthorizationConfig(), "https://app.example/cb", "state-1", "verifier-1")
	require.NoError(t, err)

	parsed, err := url.Parse(authorizeURL)
	require.NoError(t, err)
	assert.Equal(t, "github.com", parsed.Host)
	assert.Equal(t, "Iv1.client", parsed.Query().Get("client_id"))
	assert.Equal(t, "https://app.example/cb", parsed.Query().Get("redirect_uri"))
	assert.Equal(t, "state-1", parsed.Query().Get("state"))
	assert.Equal(t, PKCEChallenge("verifier-1"), parsed.Query().Get("code_challenge"))
	assert.Equal(t, "S256", parsed.Query().Get("code_challenge_method"))

	_, err = UserAuthorizationURL(config.GitHubHostedAppConfig{ID: 44, Slug: "s", PrivateKey: "p", WebhookSecret: "w"}, "cb", "s", "v")
	require.ErrorIs(t, err, ErrUserAuthorizationUnavailable)
}

func TestExchangeUserCodeReadsExpiringTokens(t *testing.T) {
	httpCtx := &queuedHTTP{responses: []*http.Response{jsonResponse(http.StatusOK, `{
		"access_token": "ghu_access",
		"expires_in": 28800,
		"refresh_token": "ghr_refresh",
		"refresh_token_expires_in": 15897600,
		"scope": "",
		"token_type": "bearer"
	}`)}}

	tokens, err := ExchangeUserCode(httpCtx, userAuthorizationConfig(), "code-1", "verifier-1", "https://app.example/cb")
	require.NoError(t, err)
	assert.Equal(t, "ghu_access", tokens.AccessToken)
	assert.Equal(t, "ghr_refresh", tokens.RefreshToken)
	require.NotNil(t, tokens.AccessTokenExpiresAt)
	assert.WithinDuration(t, time.Now().Add(8*time.Hour), *tokens.AccessTokenExpiresAt, time.Minute)

	form, err := url.ParseQuery(httpCtx.bodies[0])
	require.NoError(t, err)
	assert.Equal(t, "code-1", form.Get("code"))
	assert.Equal(t, "verifier-1", form.Get("code_verifier"))
	assert.Equal(t, "client-secret", form.Get("client_secret"))
}

func TestResolveUserAccessTokenReturnsFreshToken(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	accountID := createAuthorizedAccount(t, &UserTokens{
		AccessToken:          "ghu_fresh",
		AccessTokenExpiresAt: timeIn(time.Hour),
		RefreshToken:         "ghr_refresh",
	})

	token, err := ResolveUserAccessToken(t.Context(), database.DB(t.Context()), testEncryptor(), &queuedHTTP{}, userAuthorizationConfig(), accountID)
	require.NoError(t, err)
	assert.Equal(t, "ghu_fresh", token.Token)
	assert.Equal(t, "octocat", token.Login)

	stored, err := models.FindAccountGitHubAuthorization(database.DB(t.Context()), accountID)
	require.NoError(t, err)
	assert.NotContains(t, string(stored.EncryptedAccessToken), "ghu_fresh")
}

func TestResolveUserAccessTokenRotatesExpiringToken(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	accountID := createAuthorizedAccount(t, &UserTokens{
		AccessToken:          "ghu_old",
		AccessTokenExpiresAt: timeIn(time.Minute),
		RefreshToken:         "ghr_old",
	})
	httpCtx := &queuedHTTP{responses: []*http.Response{jsonResponse(http.StatusOK, `{
		"access_token": "ghu_new",
		"expires_in": 28800,
		"refresh_token": "ghr_new",
		"refresh_token_expires_in": 15897600
	}`)}}

	token, err := ResolveUserAccessToken(t.Context(), database.DB(t.Context()), testEncryptor(), httpCtx, userAuthorizationConfig(), accountID)
	require.NoError(t, err)
	assert.Equal(t, "ghu_new", token.Token)

	form, err := url.ParseQuery(httpCtx.bodies[0])
	require.NoError(t, err)
	assert.Equal(t, "refresh_token", form.Get("grant_type"))
	assert.Equal(t, "ghr_old", form.Get("refresh_token"))

	again, err := ResolveUserAccessToken(t.Context(), database.DB(t.Context()), testEncryptor(), &queuedHTTP{}, userAuthorizationConfig(), accountID)
	require.NoError(t, err)
	assert.Equal(t, "ghu_new", again.Token)
}

func TestResolveUserAccessTokenInvalidatesRejectedRefreshToken(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	accountID := createAuthorizedAccount(t, &UserTokens{
		AccessToken:          "ghu_old",
		AccessTokenExpiresAt: timeIn(-time.Minute),
		RefreshToken:         "ghr_revoked",
	})
	httpCtx := &queuedHTTP{responses: []*http.Response{jsonResponse(http.StatusOK, `{"error":"bad_refresh_token"}`)}}

	_, err := ResolveUserAccessToken(t.Context(), database.DB(t.Context()), testEncryptor(), httpCtx, userAuthorizationConfig(), accountID)
	require.ErrorIs(t, err, ErrUserAuthorizationNotUsable)

	stored, err := models.FindAccountGitHubAuthorization(database.DB(t.Context()), accountID)
	require.NoError(t, err)
	assert.NotNil(t, stored.InvalidatedAt)

	_, err = ResolveUserAccessToken(t.Context(), database.DB(t.Context()), testEncryptor(), &queuedHTTP{}, userAuthorizationConfig(), accountID)
	require.ErrorIs(t, err, ErrUserAuthorizationNotUsable)
}

func TestResolveUserAccessTokenRejectsTokenFromAnotherApp(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	accountID := createAuthorizedAccount(t, &UserTokens{AccessToken: "ghu_fresh", AccessTokenExpiresAt: timeIn(time.Hour)})
	otherApp := userAuthorizationConfig()
	otherApp.ID = 99

	_, err := ResolveUserAccessToken(t.Context(), database.DB(t.Context()), testEncryptor(), &queuedHTTP{}, otherApp, accountID)
	require.ErrorIs(t, err, ErrUserAuthorizationNotUsable)
}

func TestInvalidateUserAuthorizationKeepsReplacedGrant(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	accountID := createAuthorizedAccount(t, &UserTokens{AccessToken: "ghu_first", AccessTokenExpiresAt: timeIn(time.Hour)})
	db := database.DB(t.Context())
	rejected, err := ResolveUserAccessToken(t.Context(), db, testEncryptor(), &queuedHTTP{}, userAuthorizationConfig(), accountID)
	require.NoError(t, err)

	require.NoError(t, SaveUserAuthorization(t.Context(), db, testEncryptor(), accountID, userAuthorizationConfig().ID,
		&GitHubUser{ID: 583231, Login: "octocat"},
		&UserTokens{AccessToken: "ghu_second", AccessTokenExpiresAt: timeIn(time.Hour)},
	))
	require.NoError(t, InvalidateUserAuthorization(db, rejected))

	current, err := ResolveUserAccessToken(t.Context(), db, testEncryptor(), &queuedHTTP{}, userAuthorizationConfig(), accountID)
	require.NoError(t, err)
	assert.Equal(t, "ghu_second", current.Token)

	require.NoError(t, InvalidateUserAuthorization(db, current))
	_, err = ResolveUserAccessToken(t.Context(), db, testEncryptor(), &queuedHTTP{}, userAuthorizationConfig(), accountID)
	require.ErrorIs(t, err, ErrUserAuthorizationNotUsable)
}

func TestDeleteUserAuthorizationRevokesToken(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	accountID := createAuthorizedAccount(t, &UserTokens{AccessToken: "ghu_fresh", AccessTokenExpiresAt: timeIn(time.Hour)})
	httpCtx := &queuedHTTP{responses: []*http.Response{jsonResponse(http.StatusNoContent, "")}}

	revokeErr, err := DeleteUserAuthorization(t.Context(), database.DB(t.Context()), testEncryptor(), httpCtx, userAuthorizationConfig(), accountID)
	require.NoError(t, err)
	require.NoError(t, revokeErr)

	require.Len(t, httpCtx.requests, 1)
	assert.Equal(t, http.MethodDelete, httpCtx.requests[0].Method)
	assert.Equal(t, "/applications/Iv1.client/token", httpCtx.requests[0].URL.Path)
	assert.Contains(t, httpCtx.bodies[0], "ghu_fresh")

	_, err = models.FindAccountGitHubAuthorization(database.DB(t.Context()), accountID)
	require.Error(t, err)
}
