package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

func TestInstallationGitHubAppPersistence(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	tx := database.Conn()

	_, err := FindInstallationGitHubApp(tx)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	require.NoError(t, SaveInstallationGitHubApp(tx, 99, "superplane-self", []byte("pem"), []byte("whsec")))

	stored, err := FindInstallationGitHubApp(tx)
	require.NoError(t, err)
	assert.Equal(t, installationGitHubAppID, stored.ID)
	assert.Equal(t, int64(99), stored.GitHubAppID)
	assert.Equal(t, "superplane-self", stored.Slug)
	assert.Equal(t, []byte("pem"), stored.EncryptedPrivateKey)
	assert.Equal(t, []byte("whsec"), stored.EncryptedWebhookSecret)

	require.NoError(t, SaveInstallationGitHubApp(tx, 100, "superplane-next", []byte("pem-2"), []byte("whsec-2")))
	updated, err := FindInstallationGitHubApp(tx)
	require.NoError(t, err)
	assert.Equal(t, int64(100), updated.GitHubAppID)
	assert.Equal(t, "superplane-next", updated.Slug)
}
