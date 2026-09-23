package llm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestSeedDevHostedOpenRouterFromEnv(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	clearDevHostedOpenRouterEnv(t)
	t.Cleanup(func() {
		clearHostedLLMProviders(t, db)
	})

	existing, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderOpenRouter,
		Enabled:       false,
		APIKey:        []byte("keep-me"),
		ManagementKey: []byte("keep-mgmt"),
		AllowedModels: datatypes.JSONSlice[string]{"openai/gpt-4.1"},
	})
	require.NoError(t, err)

	t.Run("flag off leaves the OpenRouter row unchanged", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		t.Setenv(config.EnvDevHostedOpenRouter, "")

		require.NoError(t, llm.SeedDevHostedOpenRouterFromEnv(t.Context(), db, r.Encryptor))

		row, err := models.FindHostedLLMProvider(db, models.UsageProviderOpenRouter)
		require.NoError(t, err)
		assert.Equal(t, existing.ID, row.ID)
		assert.False(t, row.Enabled)
		assert.Equal(t, []byte("keep-me"), row.APIKey)
		assert.Equal(t, []string{"openai/gpt-4.1"}, []string(row.AllowedModels))
	})

	t.Run("production ignores the flag", func(t *testing.T) {
		setCompleteDevOpenRouterEnv(t, "production", "anthropic/claude-sonnet-4", "")

		require.NoError(t, llm.SeedDevHostedOpenRouterFromEnv(t.Context(), db, r.Encryptor))

		row, err := models.FindHostedLLMProvider(db, models.UsageProviderOpenRouter)
		require.NoError(t, err)
		assert.False(t, row.Enabled)
		assert.Equal(t, []string{"openai/gpt-4.1"}, []string(row.AllowedModels))
	})
}

func TestSeedDevHostedOpenRouter(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	t.Cleanup(func() {
		clearHostedLLMProviders(t, db)
		resetInstallationDefaultHostedModel(t, db)
	})

	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("anthropic-key"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6"},
	})
	require.NoError(t, err)

	cfg := config.DevHostedOpenRouterConfig{
		APIKey:        "sk-or",
		ManagementKey: "sk-or-mgmt",
		Models:        []string{"deepseek/deepseek-v4-flash", "deepseek/deepseek-v4-pro"},
		BaseURL:       "https://openrouter.ai/api/v1",
	}

	t.Run("writes only the OpenRouter row and fills an empty default", func(t *testing.T) {
		require.NoError(t, llm.SeedDevHostedOpenRouter(t.Context(), db, r.Encryptor, cfg))

		openrouter, err := models.FindHostedLLMProvider(db, models.UsageProviderOpenRouter)
		require.NoError(t, err)
		assert.True(t, openrouter.Enabled)
		assert.Equal(t, "https://openrouter.ai/api/v1", openrouter.BaseURL)
		assert.Equal(t, cfg.Models, []string(openrouter.AllowedModels))

		apiKey, err := llm.DecryptAPIKey(t.Context(), r.Encryptor, models.UsageProviderOpenRouter, openrouter.APIKey)
		require.NoError(t, err)
		assert.Equal(t, "sk-or", apiKey)

		managementKey, err := llm.DecryptManagementKey(t.Context(), r.Encryptor, models.UsageProviderOpenRouter, openrouter.ManagementKey)
		require.NoError(t, err)
		assert.Equal(t, "sk-or-mgmt", managementKey)

		anthropic, err := models.FindHostedLLMProvider(db, models.UsageProviderAnthropic)
		require.NoError(t, err)
		assert.Equal(t, []byte("anthropic-key"), anthropic.APIKey)
		assert.Equal(t, []string{"claude-sonnet-4-6"}, []string(anthropic.AllowedModels))

		defaultModel, err := models.GetInstallationDefaultHostedLLMModel(db)
		require.NoError(t, err)
		assert.Equal(t, models.UsageProviderOpenRouter, defaultModel.Provider)
		assert.Equal(t, "deepseek/deepseek-v4-flash", defaultModel.Model)
	})

	t.Run("does not replace an existing default unless requested", func(t *testing.T) {
		require.NoError(t, setInstallationDefaultHostedModel(db, models.UsageProviderAnthropic, "claude-sonnet-4-6"))

		require.NoError(t, llm.SeedDevHostedOpenRouter(t.Context(), db, r.Encryptor, cfg))

		defaultModel, err := models.GetInstallationDefaultHostedLLMModel(db)
		require.NoError(t, err)
		assert.Equal(t, models.UsageProviderAnthropic, defaultModel.Provider)
		assert.Equal(t, "claude-sonnet-4-6", defaultModel.Model)
	})

	t.Run("keeps an OpenRouter default that remains allowed", func(t *testing.T) {
		require.NoError(t, setInstallationDefaultHostedModel(db, models.UsageProviderOpenRouter, "deepseek/deepseek-v4-pro"))

		require.NoError(t, llm.SeedDevHostedOpenRouter(t.Context(), db, r.Encryptor, cfg))

		defaultModel, err := models.GetInstallationDefaultHostedLLMModel(db)
		require.NoError(t, err)
		assert.Equal(t, models.UsageProviderOpenRouter, defaultModel.Provider)
		assert.Equal(t, "deepseek/deepseek-v4-pro", defaultModel.Model)
	})

	t.Run("replaces an OpenRouter default that left the allowlist", func(t *testing.T) {
		withOldModel := cfg
		withOldModel.Models = []string{"openai/gpt-4.1", "deepseek/deepseek-v4-flash"}
		require.NoError(t, llm.SeedDevHostedOpenRouter(t.Context(), db, r.Encryptor, withOldModel))
		require.NoError(t, setInstallationDefaultHostedModel(db, models.UsageProviderOpenRouter, "openai/gpt-4.1"))

		require.NoError(t, llm.SeedDevHostedOpenRouter(t.Context(), db, r.Encryptor, cfg))

		defaultModel, err := models.GetInstallationDefaultHostedLLMModel(db)
		require.NoError(t, err)
		assert.Equal(t, models.UsageProviderOpenRouter, defaultModel.Provider)
		assert.Equal(t, "deepseek/deepseek-v4-flash", defaultModel.Model)
	})

	t.Run("replaces the default when SUPERPLANE_DEV_HOSTED_DEFAULT_MODEL is set", func(t *testing.T) {
		require.NoError(t, setInstallationDefaultHostedModel(db, models.UsageProviderAnthropic, "claude-sonnet-4-6"))

		withDefault := cfg
		withDefault.DefaultModel = "deepseek/deepseek-v4-pro"
		require.NoError(t, llm.SeedDevHostedOpenRouter(t.Context(), db, r.Encryptor, withDefault))

		defaultModel, err := models.GetInstallationDefaultHostedLLMModel(db)
		require.NoError(t, err)
		assert.Equal(t, models.UsageProviderOpenRouter, defaultModel.Provider)
		assert.Equal(t, "deepseek/deepseek-v4-pro", defaultModel.Model)
	})
}

func setCompleteDevOpenRouterEnv(t *testing.T, appEnv, modelsCSV, defaultModel string) {
	t.Helper()
	t.Setenv("APP_ENV", appEnv)
	t.Setenv(config.EnvDevHostedOpenRouter, "yes")
	t.Setenv(config.EnvDevOpenRouterAPIKey, "sk-or")
	t.Setenv(config.EnvDevOpenRouterManagementKey, "sk-or-mgmt")
	t.Setenv(config.EnvDevOpenRouterModels, modelsCSV)
	t.Setenv(config.EnvDevOpenRouterBaseURL, "")
	t.Setenv(config.EnvDevHostedDefaultModel, defaultModel)
}

func clearDevHostedOpenRouterEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "")
	t.Setenv(config.EnvDevHostedOpenRouter, "")
	t.Setenv(config.EnvDevOpenRouterAPIKey, "")
	t.Setenv(config.EnvDevOpenRouterManagementKey, "")
	t.Setenv(config.EnvDevOpenRouterModels, "")
	t.Setenv(config.EnvDevOpenRouterBaseURL, "")
	t.Setenv(config.EnvDevHostedDefaultModel, "")
}

func clearHostedLLMProviders(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Where("provider <> ?", "").Delete(&models.HostedLLMProvider{}).Error)
}

func resetInstallationDefaultHostedModel(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, setInstallationDefaultHostedModel(db, "", ""))
}

func setInstallationDefaultHostedModel(db *gorm.DB, provider, model string) error {
	settings, err := models.GetInstallationLLMSettings(db)
	if err != nil {
		return err
	}
	next := *settings
	next.DefaultHostedProvider = stringPointerOrNil(provider)
	next.DefaultHostedModel = stringPointerOrNil(model)
	_, err = models.UpdateInstallationLLMSettings(db, next)
	return err
}

func stringPointerOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
