package seed

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("SEED_EMAIL", "")
	t.Setenv("SEED_PASSWORD", "")
	t.Setenv("SEED_NAME", "")
	t.Setenv("SEED_ORG", "")
	t.Setenv("SEED_WORKSPACE", "")
	t.Setenv("BASE_URL", "")
	t.Setenv("ANTHROPIC_API_KEY", "sk-test")
	t.Setenv(config.EnvGitHubAppID, "123")
	t.Setenv(config.EnvGitHubAppSlug, "superplane-dev")
	t.Setenv(config.EnvGitHubAppPrivateKey, "-----BEGIN RSA PRIVATE KEY-----\nMIIBOgIBAAJBAK8=\n-----END RSA PRIVATE KEY-----")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "whsec")

	cfg := LoadConfig()
	assert.Equal(t, defaultEmail, cfg.Email)
	assert.Equal(t, defaultPassword, cfg.Password)
	assert.Equal(t, defaultName, cfg.Name)
	assert.Equal(t, defaultOrganization, cfg.OrganizationName)
	assert.Equal(t, defaultWorkspace, cfg.WorkspaceName)
	assert.Equal(t, "http://localhost:8000", cfg.BaseURL)
	require.NoError(t, cfg.Validate())
}

func TestConfigValidateMissingGitHubApp(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "")
	t.Setenv(config.EnvGitHubAppSlug, "")
	t.Setenv(config.EnvGitHubAppPrivateKey, "")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "")
	t.Setenv("ANTHROPIC_API_KEY", "sk-test")

	cfg := LoadConfig()
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), config.EnvGitHubAppID)
}

func TestConfigValidateMissingAnthropicKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv(config.EnvGitHubAppID, "123")
	t.Setenv(config.EnvGitHubAppSlug, "superplane-dev")
	t.Setenv(config.EnvGitHubAppPrivateKey, "-----BEGIN RSA PRIVATE KEY-----\nMIIBOgIBAAJBAK8=\n-----END RSA PRIVATE KEY-----")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "whsec")

	cfg := LoadConfig()
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ANTHROPIC_API_KEY")
}
