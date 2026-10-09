package public

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestHandleGitHubAppManifest(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "")
	t.Setenv(config.EnvGitHubAppSlug, "")
	t.Setenv(config.EnvGitHubAppPrivateKey, "")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "")
	r := support.Setup(t)
	server, account, token := setupTestServer(r, t)
	server.BaseURL = "https://app.example"
	server.WebhooksBaseURL = "https://hooks.example"

	t.Run("rejects create when the account is not an installation admin", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     http.MethodGet,
			path:       "/github/app/manifest?return_to=/org/workspaces/new/setup",
			authCookie: token,
		})
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	require.NoError(t, models.PromoteToInstallationAdmin(account.ID.String()))

	t.Run("returns the GitHub App create form", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     http.MethodGet,
			path:       "/github/app/manifest?return_to=/org/workspaces/new/setup",
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)

		var body githubAppManifestResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.Equal(t, "https://github.com/settings/apps/new", body.URL)
		assert.Equal(t, http.MethodPost, body.Method)
		assert.Contains(t, body.Form["manifest"], `"public":false`)
		assert.Contains(t, body.Form["manifest"], "https://hooks.example/api/v1/github/app/webhook")
		returnPath, accountID, err := githubapp.VerifyCreateState("test-client-secret", body.Form["state"])
		require.NoError(t, err)
		assert.Equal(t, "/org/workspaces/new/setup", returnPath)
		assert.Equal(t, account.ID, accountID)
	})

	t.Run("rejects create when env already holds the app", func(t *testing.T) {
		t.Setenv(config.EnvGitHubAppID, "99")
		t.Setenv(config.EnvGitHubAppSlug, "superplane")
		t.Setenv(config.EnvGitHubAppPrivateKey, "pem")
		t.Setenv(config.EnvGitHubAppWebhookSecret, "whsec")
		response := execRequest(server, requestParams{
			method:     http.MethodGet,
			path:       "/github/app/manifest",
			authCookie: token,
		})
		assert.Equal(t, http.StatusConflict, response.Code)
	})
}

func TestHandleGitHubAppCreatedStartsAccountConnection(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "")
	t.Setenv(config.EnvGitHubAppSlug, "")
	t.Setenv(config.EnvGitHubAppPrivateKey, "")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "")
	r := support.Setup(t)
	server, account, _ := setupTestServer(r, t)
	require.NoError(t, models.PromoteToInstallationAdmin(account.ID.String()))
	require.NoError(t, githubapp.Save(t.Context(), database.DB(t.Context()), r.Encryptor, config.GitHubHostedAppConfig{
		ID:            44,
		Slug:          "superplane-self",
		PrivateKey:    "pem",
		WebhookSecret: "whsec",
		ClientID:      "Iv1.installation",
		ClientSecret:  "installation-secret",
	}))
	state, err := githubapp.SignCreateState("test-client-secret", "/demo/workspaces/new/setup?step=vcs", account.ID)
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method: http.MethodGet,
		path:   "/github/app/created?code=abc&state=" + state,
	})

	require.Equal(t, http.StatusFound, response.Code)
	location, err := url.Parse(response.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "/auth/github", location.Path)
	assert.Equal(t, "connect", location.Query().Get("intent"))
	redirect, err := url.Parse(location.Query().Get("redirect"))
	require.NoError(t, err)
	assert.Equal(t, "/demo/workspaces/new/setup", redirect.Path)
	assert.Equal(t, "1", redirect.Query().Get("githubConnected"))
}

func TestHandleGitHubAppCreatedKeepsAppThatNeedsLoginClient(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "")
	t.Setenv(config.EnvGitHubAppSlug, "")
	t.Setenv(config.EnvGitHubAppPrivateKey, "")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "")
	r := support.Setup(t)
	server, account, _ := setupTestServer(r, t)
	t.Setenv(config.EnvGitHubAppID, "")
	t.Setenv(config.EnvGitHubAppSlug, "")
	t.Setenv(config.EnvGitHubAppPrivateKey, "")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "")
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	require.NoError(t, models.PromoteToInstallationAdmin(account.ID.String()))
	require.NoError(t, githubapp.Save(t.Context(), database.DB(t.Context()), r.Encryptor, config.GitHubHostedAppConfig{
		ID:            44,
		Slug:          "superplane-self",
		PrivateKey:    "pem",
		WebhookSecret: "whsec",
	}))
	state, err := githubapp.SignCreateState("test-client-secret", "/demo/workspaces/new/setup?step=vcs", account.ID)
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method: http.MethodGet,
		path:   "/github/app/created?code=abc&state=" + state,
	})

	require.Equal(t, http.StatusFound, response.Code)
	location, err := url.Parse(response.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "/demo/workspaces/new/setup", location.Path)
	assert.Equal(t, "needs_client", location.Query().Get("githubLogin"))
	stored, err := models.FindInstallationGitHubApp(database.DB(t.Context()))
	require.NoError(t, err)
	assert.Equal(t, int64(44), stored.GitHubAppID)
	assert.Empty(t, stored.ClientID)
}

func TestHandleGitHubAppLoginSavesClientOnExistingApp(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "")
	t.Setenv(config.EnvGitHubAppSlug, "")
	t.Setenv(config.EnvGitHubAppPrivateKey, "")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "")
	r := support.Setup(t)
	server, account, token := setupTestServer(r, t)
	t.Setenv(config.EnvGitHubAppID, "")
	t.Setenv(config.EnvGitHubAppSlug, "")
	t.Setenv(config.EnvGitHubAppPrivateKey, "")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "")
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	require.NoError(t, models.PromoteToInstallationAdmin(account.ID.String()))

	t.Run("rejects login client when no GitHub App exists", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     http.MethodPost,
			path:       "/github/app/login",
			body:       []byte(`{"clientId":"Iv1.client","clientSecret":"secret"}`),
			authCookie: token,
		})
		assert.Equal(t, http.StatusConflict, response.Code)
	})

	require.NoError(t, githubapp.Save(t.Context(), database.DB(t.Context()), r.Encryptor, config.GitHubHostedAppConfig{
		ID:            44,
		Slug:          "superplane-self",
		PrivateKey:    "pem",
		WebhookSecret: "whsec",
	}))

	t.Run("reports that the existing app needs a login client", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     http.MethodGet,
			path:       "/github/app/login",
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)
		var body githubAppLoginResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.Equal(t, githubapp.LoginClientNeeded, body.State)
		assert.Equal(t, "superplane-self", body.Slug)
	})

	t.Run("stores the login client without replacing the app", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     http.MethodPost,
			path:       "/github/app/login",
			body:       []byte(`{"clientId":"Iv1.client","clientSecret":"secret"}`),
			authCookie: token,
		})
		require.Equal(t, http.StatusNoContent, response.Code)
		cfg, err := githubapp.Resolve(t.Context(), database.DB(t.Context()), r.Encryptor)
		require.NoError(t, err)
		assert.Equal(t, int64(44), cfg.ID)
		assert.Equal(t, "pem", cfg.PrivateKey)
		assert.Equal(t, "Iv1.client", cfg.ClientID)
		assert.Equal(t, "secret", cfg.ClientSecret)
	})

	t.Run("replaces a wrong login client without replacing the app", func(t *testing.T) {
		read := execRequest(server, requestParams{
			method:     http.MethodGet,
			path:       "/github/app/login",
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, read.Code)
		var body githubAppLoginResponse
		require.NoError(t, json.Unmarshal(read.Body.Bytes(), &body))
		assert.Equal(t, githubapp.LoginClientReady, body.State)
		assert.True(t, body.CanUpdate)

		response := execRequest(server, requestParams{
			method:     http.MethodPost,
			path:       "/github/app/login",
			body:       []byte(`{"clientId":"Iv1.other","clientSecret":"corrected"}`),
			authCookie: token,
		})
		require.Equal(t, http.StatusNoContent, response.Code)
		cfg, err := githubapp.Resolve(t.Context(), database.DB(t.Context()), r.Encryptor)
		require.NoError(t, err)
		assert.Equal(t, int64(44), cfg.ID)
		assert.Equal(t, "pem", cfg.PrivateKey)
		assert.Equal(t, "whsec", cfg.WebhookSecret)
		assert.Equal(t, "Iv1.other", cfg.ClientID)
		assert.Equal(t, "corrected", cfg.ClientSecret)
	})
}

func TestHandleGitHubAppCreatedRejectsInvalidState(t *testing.T) {
	r := support.Setup(t)
	server, _, _ := setupTestServer(r, t)

	response := execRequest(server, requestParams{
		method: http.MethodGet,
		path:   "/github/app/created?code=abc&state=bad",
	})
	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestHandleGitHubAppCreatedRequiresInstallationAdmin(t *testing.T) {
	r := support.Setup(t)
	server, account, _ := setupTestServer(r, t)

	t.Run("rejects a signed state from a non-admin account", func(t *testing.T) {
		state, err := githubapp.SignCreateState("test-client-secret", "/org/workspaces/new/setup", account.ID)
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method: http.MethodGet,
			path:   "/github/app/created?code=abc&state=" + state,
		})
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	t.Run("rejects a signed state after the admin is demoted", func(t *testing.T) {
		require.NoError(t, models.PromoteToInstallationAdmin(account.ID.String()))
		state, err := githubapp.SignCreateState("test-client-secret", "/org/workspaces/new/setup", account.ID)
		require.NoError(t, err)
		require.NoError(t, models.DemoteFromInstallationAdmin(account.ID.String()))
		response := execRequest(server, requestParams{
			method: http.MethodGet,
			path:   "/github/app/created?code=abc&state=" + state,
		})
		assert.Equal(t, http.StatusForbidden, response.Code)
	})
}
