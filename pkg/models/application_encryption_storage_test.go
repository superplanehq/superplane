package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestApplicationEncryptedStorageHandlersMatchSchema(t *testing.T) {
	db := database.DB(t.Context())
	require.NoError(t, database.VerifyTestDatabase(db))

	for _, location := range ApplicationEncryptedStorage() {
		t.Run(location.Name, func(t *testing.T) {
			_, err := location.ReadPage(db, "", 1)
			require.NoError(t, err)

			cursor := uuid.NewString()
			switch location.Name {
			case "installation_github_apps.encrypted_private_key", "installation_github_apps.encrypted_webhook_secret", "installation_github_apps.encrypted_client_secret", "installation_licenses.encrypted_license":
				cursor = "0"
			case "bitbucket_forge_installations.system_token", "sentry_app_install_grants.access_token", "sentry_app_install_grants.refresh_token":
				cursor = "encryption-test-" + cursor
			}
			oldValue := []byte("old")
			if location.Format == EncryptedStorageIntegrationJSON {
				oldValue = []byte("{}")
			}
			changed, err := location.Replace(db, EncryptedStorageRow{Cursor: cursor, Value: oldValue}, oldValue)
			require.NoError(t, err)
			require.False(t, changed)
		})
	}
}
