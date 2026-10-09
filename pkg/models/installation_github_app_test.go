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
	tx := database.DB(t.Context())

	_, err := FindInstallationGitHubApp(tx)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	require.NoError(t, SaveInstallationGitHubApp(tx, 99, "superplane-self", "client-id", []byte("pem"), []byte("whsec"), []byte("client-secret")))

	stored, err := FindInstallationGitHubApp(tx)
	require.NoError(t, err)
	assert.Equal(t, installationGitHubAppID, stored.ID)
	assert.Equal(t, int64(99), stored.GitHubAppID)
	assert.Equal(t, "superplane-self", stored.Slug)
	assert.Equal(t, "client-id", stored.ClientID)
	assert.Equal(t, []byte("pem"), stored.EncryptedPrivateKey)
	assert.Equal(t, []byte("whsec"), stored.EncryptedWebhookSecret)
	assert.Equal(t, []byte("client-secret"), stored.EncryptedClientSecret)

	err = SaveInstallationGitHubApp(tx, 100, "superplane-next", "next-client", []byte("pem-2"), []byte("whsec-2"), []byte("next-secret"))
	require.ErrorIs(t, err, ErrInstallationGitHubAppExists)
	kept, err := FindInstallationGitHubApp(tx)
	require.NoError(t, err)
	assert.Equal(t, int64(99), kept.GitHubAppID)
	assert.Equal(t, "superplane-self", kept.Slug)
	assert.Equal(t, []byte("pem"), kept.EncryptedPrivateKey)
}
