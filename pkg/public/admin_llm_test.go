package public

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestAdminLLMSettings(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	_, err := models.UpdateInstallationLLMSettings(database.Conn(), models.InstallationLLMSettings{
		WelcomeGrantCents:   models.DefaultWelcomeGrantCents,
		MarkupBPS:           models.DefaultMarkupBPS,
		WarningThresholdBPS: models.DefaultWarningThresholdBPS,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = models.UpdateInstallationLLMSettings(database.Conn(), models.InstallationLLMSettings{
			WelcomeGrantCents:   models.DefaultWelcomeGrantCents,
			MarkupBPS:           models.DefaultMarkupBPS,
			WarningThresholdBPS: models.DefaultWarningThresholdBPS,
		})
		_ = database.Conn().Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
	})

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-llm@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/installation/llm-settings",
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("admin can load defaults and update welcome grant", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/installation/llm-settings",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var settings installationLLMSettingsResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &settings))
		assert.Equal(t, models.DefaultWelcomeGrantCents, settings.WelcomeGrantCents)
		assert.Equal(t, models.DefaultMarkupBPS, settings.MarkupBPS)
		require.Len(t, settings.Providers, 3)

		body, err := json.Marshal(map[string]any{
			"welcome_grant_cents": 2500,
			"markup_bps":          1000,
		})
		require.NoError(t, err)
		response = execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/installation/llm-settings",
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &settings))
		assert.Equal(t, int64(2500), settings.WelcomeGrantCents)
		assert.Equal(t, 1000, settings.MarkupBPS)
	})

	t.Run("admin can save a hosted provider", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"enabled":        true,
			"api_key":        "sk-test",
			"allowed_models": []string{"claude-sonnet-4-6"},
		})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/installation/llm-providers/anthropic",
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var settings installationLLMSettingsResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &settings))
		var anthropic hostedLLMProviderResponse
		for _, provider := range settings.Providers {
			if provider.Provider == "anthropic" {
				anthropic = provider
			}
		}
		assert.True(t, anthropic.Enabled)
		assert.True(t, anthropic.APIKeyConfigured)
		assert.Equal(t, []string{"claude-sonnet-4-6"}, anthropic.AllowedModels)
	})

	t.Run("admin can grant credit and set markup override", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/llm-credit",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var credit organizationLLMCreditResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &credit))
		assert.Greater(t, credit.RemainingCreditCents, int64(0))

		body, err := json.Marshal(map[string]any{
			"amount_cents": 1000,
			"note":         "restore",
		})
		require.NoError(t, err)
		response = execRequest(server, requestParams{
			method:      "POST",
			path:        "/admin/api/organizations/" + r.Organization.ID.String() + "/llm-credit/grants",
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &credit))
		assert.GreaterOrEqual(t, credit.GrantTotalCents, int64(6000))

		body, err = json.Marshal(map[string]any{"markup_bps": 0})
		require.NoError(t, err)
		response = execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/organizations/" + r.Organization.ID.String() + "/llm-settings",
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &credit))
		require.NotNil(t, credit.MarkupOverrideBPS)
		assert.Equal(t, 0, *credit.MarkupOverrideBPS)
	})
}
