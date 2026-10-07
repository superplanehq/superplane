package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDevHostedOpenRouterConfig(t *testing.T) {
	clearDevHostedOpenRouterEnv(t)

	t.Run("unset env is not enabled", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")

		cfg, err := LoadDevHostedOpenRouterConfig()
		require.NoError(t, err)
		assert.False(t, cfg.Enabled())
	})

	t.Run("flag without keys returns an error", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		t.Setenv(EnvDevHostedOpenRouter, "yes")

		cfg, err := LoadDevHostedOpenRouterConfig()
		require.Error(t, err)
		assert.False(t, cfg.Enabled())
		assert.Contains(t, err.Error(), "SUPERPLANE_DEV_OPENROUTER_API_KEY")
	})

	t.Run("complete set is enabled", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		t.Setenv(EnvDevHostedOpenRouter, "yes")
		t.Setenv(EnvDevOpenRouterAPIKey, "sk-or")
		t.Setenv(EnvDevOpenRouterManagementKey, "sk-or-mgmt")
		t.Setenv(EnvDevOpenRouterModels, " deepseek/deepseek-v4-flash , deepseek/deepseek-v4-pro ")
		t.Setenv(EnvDevOpenRouterBaseURL, "https://openrouter.example/api/v1")
		t.Setenv(EnvDevHostedDefaultModel, "deepseek/deepseek-v4-pro")

		cfg, err := LoadDevHostedOpenRouterConfig()
		require.NoError(t, err)
		assert.True(t, cfg.Enabled())
		assert.Equal(t, "sk-or", cfg.APIKey)
		assert.Equal(t, "sk-or-mgmt", cfg.ManagementKey)
		assert.Equal(t, []string{"deepseek/deepseek-v4-flash", "deepseek/deepseek-v4-pro"}, cfg.Models)
		assert.Equal(t, "https://openrouter.example/api/v1", cfg.BaseURL)
		assert.Equal(t, "deepseek/deepseek-v4-pro", cfg.DefaultModel)
	})

	t.Run("production ignores the flag", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv(EnvDevHostedOpenRouter, "yes")
		t.Setenv(EnvDevOpenRouterAPIKey, "sk-or")
		t.Setenv(EnvDevOpenRouterManagementKey, "sk-or-mgmt")
		t.Setenv(EnvDevOpenRouterModels, "openai/gpt-5")

		cfg, err := LoadDevHostedOpenRouterConfig()
		require.NoError(t, err)
		assert.False(t, cfg.Enabled())
	})

	t.Run("default model must be on the allowlist", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")
		t.Setenv(EnvDevHostedOpenRouter, "yes")
		t.Setenv(EnvDevOpenRouterAPIKey, "sk-or")
		t.Setenv(EnvDevOpenRouterManagementKey, "sk-or-mgmt")
		t.Setenv(EnvDevOpenRouterModels, "openai/gpt-5")
		t.Setenv(EnvDevHostedDefaultModel, "anthropic/claude-sonnet-4")

		cfg, err := LoadDevHostedOpenRouterConfig()
		require.Error(t, err)
		assert.False(t, cfg.Enabled())
		assert.Contains(t, err.Error(), "SUPERPLANE_DEV_HOSTED_DEFAULT_MODEL")
	})
}

func clearDevHostedOpenRouterEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "")
	t.Setenv(EnvDevHostedOpenRouter, "")
	t.Setenv(EnvDevOpenRouterAPIKey, "")
	t.Setenv(EnvDevOpenRouterManagementKey, "")
	t.Setenv(EnvDevOpenRouterModels, "")
	t.Setenv(EnvDevOpenRouterBaseURL, "")
	t.Setenv(EnvDevHostedDefaultModel, "")
}
