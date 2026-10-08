package githubapp

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func testEncryptor() crypto.Encryptor {
	return crypto.NewAESGCMEncryptor([]byte("0123456789abcdef0123456789abcdef"))
}

func TestResolvePrefersEnvironment(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	encryptor := testEncryptor()
	clearGitHubAppEnv(t)
	require.NoError(t, Save(t.Context(), database.DB(t.Context()), encryptor, config.GitHubHostedAppConfig{
		ID:            11,
		Slug:          "from-db",
		PrivateKey:    "db-pem",
		WebhookSecret: "db-secret",
	}))
	t.Setenv(config.EnvGitHubAppID, "22")
	t.Setenv(config.EnvGitHubAppSlug, "from-env")
	t.Setenv(config.EnvGitHubAppPrivateKey, "env-pem")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "env-secret")

	cfg, err := Resolve(t.Context(), database.DB(t.Context()), encryptor)
	require.NoError(t, err)
	assert.Equal(t, int64(22), cfg.ID)
	assert.Equal(t, "from-env", cfg.Slug)
	assert.Equal(t, "env-pem", cfg.PrivateKey)
	assert.Equal(t, "env-secret", cfg.WebhookSecret)
}

func TestResolveReadsInstallationRow(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	encryptor := testEncryptor()
	clearGitHubAppEnv(t)
	require.NoError(t, Save(t.Context(), database.DB(t.Context()), encryptor, config.GitHubHostedAppConfig{
		ID:            11,
		Slug:          "from-db",
		PrivateKey:    "db-pem",
		WebhookSecret: "db-secret",
	}))

	cfg, err := Resolve(t.Context(), database.DB(t.Context()), encryptor)
	require.NoError(t, err)
	assert.True(t, cfg.Enabled())
	assert.Equal(t, int64(11), cfg.ID)
	assert.Equal(t, "from-db", cfg.Slug)
	assert.Equal(t, "db-pem", cfg.PrivateKey)
	assert.Equal(t, "db-secret", cfg.WebhookSecret)

	record, err := models.FindInstallationGitHubApp(database.DB(t.Context()))
	require.NoError(t, err)
	assert.False(t, bytes.Equal(record.EncryptedPrivateKey, []byte("db-pem")))
	assert.False(t, bytes.Equal(record.EncryptedWebhookSecret, []byte("db-secret")))
}

func TestResolveEmptyWhenMissing(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	clearGitHubAppEnv(t)

	cfg, err := Resolve(t.Context(), database.DB(t.Context()), testEncryptor())
	require.NoError(t, err)
	assert.False(t, cfg.Enabled())
}

func clearGitHubAppEnv(t *testing.T) {
	t.Helper()
	t.Setenv(config.EnvGitHubAppID, "")
	t.Setenv(config.EnvGitHubAppSlug, "")
	t.Setenv(config.EnvGitHubAppPrivateKey, "")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "")
}
