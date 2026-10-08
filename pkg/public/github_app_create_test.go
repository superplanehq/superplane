package public

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/test/support"
)

func TestHandleGitHubAppManifest(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "")
	t.Setenv(config.EnvGitHubAppSlug, "")
	t.Setenv(config.EnvGitHubAppPrivateKey, "")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "")
	r := support.Setup(t)
	server, _, token := setupTestServer(r, t)
	server.BaseURL = "https://app.example"
	server.WebhooksBaseURL = "https://hooks.example"

	t.Run("returns the public app create form", func(t *testing.T) {
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
		assert.Contains(t, body.Form["manifest"], `"public":true`)
		assert.Contains(t, body.Form["manifest"], "https://hooks.example/api/v1/github/app/webhook")
		returnPath, err := githubapp.VerifyCreateState("test-client-secret", body.Form["state"])
		require.NoError(t, err)
		assert.Equal(t, "/org/workspaces/new/setup", returnPath)
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
