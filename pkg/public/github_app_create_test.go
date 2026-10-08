package public

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
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
