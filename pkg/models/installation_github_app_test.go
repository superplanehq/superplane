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

	require.NoError(t, SaveInstallationGitHubApp(tx, InstallationGitHubApp{
		GitHubAppID:            99,
		Slug:                   "superplane-self",
		EncryptedPrivateKey:    []byte("pem"),
		EncryptedWebhookSecret: []byte("whsec"),
		ClientID:               "Iv1.self",
		EncryptedClientSecret:  []byte("client-secret"),
	}))

	stored, err := FindInstallationGitHubApp(tx)
	require.NoError(t, err)
	assert.Equal(t, installationGitHubAppID, stored.ID)
	assert.Equal(t, int64(99), stored.GitHubAppID)
	assert.Equal(t, "superplane-self", stored.Slug)
	assert.Equal(t, []byte("pem"), stored.EncryptedPrivateKey)
	assert.Equal(t, []byte("whsec"), stored.EncryptedWebhookSecret)
	assert.Equal(t, "Iv1.self", stored.ClientID)
	assert.Equal(t, []byte("client-secret"), stored.EncryptedClientSecret)

	err = SaveInstallationGitHubApp(tx, InstallationGitHubApp{
		GitHubAppID:            100,
		Slug:                   "superplane-next",
		EncryptedPrivateKey:    []byte("pem-2"),
		EncryptedWebhookSecret: []byte("whsec-2"),
	})
	require.ErrorIs(t, err, ErrInstallationGitHubAppExists)
	kept, err := FindInstallationGitHubApp(tx)
	require.NoError(t, err)
	assert.Equal(t, int64(99), kept.GitHubAppID)
	assert.Equal(t, "superplane-self", kept.Slug)
	assert.Equal(t, []byte("pem"), kept.EncryptedPrivateKey)
}
