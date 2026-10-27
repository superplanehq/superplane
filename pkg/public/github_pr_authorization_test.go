package public

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/pkg/models"
)

func clearGitHubAppEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		config.EnvGitHubAppID,
		config.EnvGitHubAppSlug,
		config.EnvGitHubAppPrivateKey,
		config.EnvGitHubAppWebhookSecret,
		config.EnvGitHubAppClientID,
		config.EnvGitHubAppClientSecret,
	} {
		t.Setenv(name, "")
	}
}

func saveUserAuthorizationApp(t *testing.T, server *Server) {
	t.Helper()
	require.NoError(t, githubapp.Save(t.Context(), database.DB(t.Context()), server.encryptor, config.GitHubHostedAppConfig{
		ID:            44,
		Slug:          "superplane",
		PrivateKey:    "pem",
		WebhookSecret: "whsec",
		ClientID:      "Iv1.client",
		ClientSecret:  "client-secret",
	}))
}

func saveGitHubPRAuthorization(t *testing.T, server *Server, accountID uuid.UUID, appID int64) {
	t.Helper()
	expiresAt := time.Now().Add(time.Hour)
	require.NoError(t, githubapp.SaveUserAuthorization(
		t.Context(),
		database.DB(t.Context()),
		server.encryptor,
		accountID,
		appID,
		&githubapp.GitHubUser{ID: 4242, Login: "ada-gh"},
		&githubapp.UserTokens{AccessToken: "ghu_token", AccessTokenExpiresAt: &expiresAt},
	))
}

func githubPRAuthorizationCallback(server *Server, token, state, verifier, query string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/account/github-authorization/callback?state="+url.QueryEscape(state)+query, nil)
	request.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	if verifier != "" {
		request.AddCookie(&http.Cookie{Name: githubPRAuthorizationCookie, Value: verifier})
	}
	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, request)
	return response
}

func TestStartGitHubPRAuthorization(t *testing.T) {
	clearGitHubAppEnv(t)
	server, registry, token := newLinkedAccountServer(t)
	server.BaseURL = "https://app.example"

	t.Run("rejects start when the app has no client credentials", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     http.MethodGet,
			path:       "/account/github-authorization/start",
			authCookie: token,
		})
		assert.Equal(t, http.StatusServiceUnavailable, response.Code)
	})

	saveUserAuthorizationApp(t, server)

	t.Run("redirects to GitHub with state bound to the account and verifier cookie", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     http.MethodGet,
			path:       "/account/github-authorization/start?redirect=/acme/settings/profile",
			authCookie: token,
		})
		require.Equal(t, http.StatusFound, response.Code)

		location, err := url.Parse(response.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "github.com", location.Host)
		assert.Equal(t, "Iv1.client", location.Query().Get("client_id"))
		assert.Equal(t, "https://app.example/account/github-authorization/callback", location.Query().Get("redirect_uri"))

		cookies := response.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, githubPRAuthorizationCookie, cookies[0].Name)
		assert.True(t, cookies[0].HttpOnly)
		assert.True(t, cookies[0].Secure)
		assert.Equal(t, githubapp.PKCEChallenge(cookies[0].Value), location.Query().Get("code_challenge"))

		state, err := server.parseGitHubPRAuthorizationState(location.Query().Get("state"))
		require.NoError(t, err)
		assert.Equal(t, registry.Account.ID, state.AccountID)
		assert.Equal(t, "/acme/settings/profile", state.Redirect)
	})
}

func TestCompleteGitHubPRAuthorizationRejectsUnboundCallbacks(t *testing.T) {
	clearGitHubAppEnv(t)
	server, registry, token := newLinkedAccountServer(t)
	saveUserAuthorizationApp(t, server)
	verifier, err := githubapp.NewPKCEVerifier()
	require.NoError(t, err)

	t.Run("rejects state for another account", func(t *testing.T) {
		state, err := server.signGitHubPRAuthorizationState(uuid.New(), "/", githubapp.PKCEChallenge(verifier))
		require.NoError(t, err)

		response := githubPRAuthorizationCallback(server, token, state, verifier, "&code=abc")
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	t.Run("rejects a callback without the browser verifier", func(t *testing.T) {
		state, err := server.signGitHubPRAuthorizationState(registry.Account.ID, "/", githubapp.PKCEChallenge(verifier))
		require.NoError(t, err)

		response := githubPRAuthorizationCallback(server, token, state, "other-verifier", "&code=abc")
		assert.Equal(t, http.StatusBadRequest, response.Code)

		response = githubPRAuthorizationCallback(server, token, state, "", "&code=abc")
		assert.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("returns to the profile when the member cancels on GitHub", func(t *testing.T) {
		state, err := server.signGitHubPRAuthorizationState(registry.Account.ID, "/acme/settings/profile", githubapp.PKCEChallenge(verifier))
		require.NoError(t, err)

		response := githubPRAuthorizationCallback(server, token, state, verifier, "&error=access_denied")
		require.Equal(t, http.StatusSeeOther, response.Code)
		location, err := url.Parse(response.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "/acme/settings/profile", location.Path)
		assert.Equal(t, githubPRAuthorizationErrorDenied, location.Query().Get("auth_error"))
	})

	_, err = models.FindAccountGitHubAuthorization(database.DB(t.Context()), registry.Account.ID)
	require.Error(t, err)
}

func TestGetAccountReportsGitHubPRAuthorization(t *testing.T) {
	clearGitHubAppEnv(t)
	server, registry, token := newLinkedAccountServer(t)

	readAuthorization := func() AccountGitHubPRAuthorizationResponse {
		response := execRequest(server, requestParams{method: http.MethodGet, path: "/account", authCookie: token})
		require.Equal(t, http.StatusOK, response.Code)
		var body AccountResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		return body.GitHubPRAuthorization
	}

	assert.Equal(t, AccountGitHubPRAuthorizationResponse{}, readAuthorization())

	saveUserAuthorizationApp(t, server)
	assert.Equal(t, AccountGitHubPRAuthorizationResponse{Available: true}, readAuthorization())

	saveGitHubPRAuthorization(t, server, registry.Account.ID, 44)
	assert.Equal(t, AccountGitHubPRAuthorizationResponse{Available: true, Enabled: true, Login: "ada-gh"}, readAuthorization())

	saveGitHubPRAuthorization(t, server, registry.Account.ID, 99)
	assert.Equal(t, AccountGitHubPRAuthorizationResponse{Available: true, Login: "ada-gh", NeedsReauthorization: true}, readAuthorization())
}

func TestDisableGitHubPRAuthorizationRemovesGrant(t *testing.T) {
	clearGitHubAppEnv(t)
	server, registry, token := newLinkedAccountServer(t)
	saveGitHubPRAuthorization(t, server, registry.Account.ID, 44)

	response := execRequest(server, requestParams{
		method:     http.MethodDelete,
		path:       "/account/github-authorization",
		authCookie: token,
	})

	require.Equal(t, http.StatusNoContent, response.Code)
	_, err := models.FindAccountGitHubAuthorization(database.DB(t.Context()), registry.Account.ID)
	require.Error(t, err)
}

func TestDisconnectLinkedGitHubAccountRemovesPRAuthorization(t *testing.T) {
	clearGitHubAppEnv(t)
	server, registry, token := newLinkedAccountServer(t)
	saveLinkedAccount(t, registry, models.ProviderGitHub, "4242", "ada-gh")
	saveGitHubPRAuthorization(t, server, registry.Account.ID, 44)

	response := deleteLinkedAccount(t, server, token, models.ProviderGitHub, "4242")

	require.Equal(t, http.StatusNoContent, response.Code)
	_, err := models.FindAccountGitHubAuthorization(database.DB(t.Context()), registry.Account.ID)
	require.Error(t, err)
}
