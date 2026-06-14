package contexts

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__IntegrationSecretStorage(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"dummy",
		support.RandomName("installation"),
		map[string]any{},
	)
	require.NoError(t, err)

	t.Run("loads and decrypts existing secrets", func(t *testing.T) {
		seedContextIntegrationSecret(t, integration, "token", "initial-token")

		storage := NewIntegrationSecretStorage(database.Conn(), crypto.NewNoOpEncryptor(), integration)
		value, err := storage.Get("token")
		require.NoError(t, err)
		assert.Equal(t, "initial-token", value)

		value, err = storage.Get("missing")
		require.Error(t, err)
		assert.Empty(t, value)
		assert.Contains(t, err.Error(), "secret missing not found")
	})

	t.Run("creates and persists secrets", func(t *testing.T) {
		storage := NewIntegrationSecretStorage(database.Conn(), crypto.NewNoOpEncryptor(), integration)
		err := storage.Create(core.IntegrationSecretDefinition{
			Name:        "created-token",
			Label:       "Created token",
			Description: "created by test",
			Value:       "new-secret",
			Editable:    true,
		})
		require.NoError(t, err)

		value, err := storage.Get("created-token")
		require.NoError(t, err)
		assert.Equal(t, "new-secret", value)

		var secret models.IntegrationSecret
		err = database.Conn().
			Where("installation_id = ? AND name = ?", integration.ID, "created-token").
			First(&secret).
			Error
		require.NoError(t, err)
		assert.Equal(t, r.Organization.ID, secret.OrganizationID)
		assert.Equal(t, "Created token", secret.Label)
		assert.Equal(t, "created by test", secret.Description)
		assert.Equal(t, []byte("new-secret"), secret.Value)
		assert.True(t, secret.Editable)
		assert.NotNil(t, secret.CreatedAt)
		assert.NotNil(t, secret.UpdatedAt)
	})

	t.Run("rejects empty and duplicate secret names", func(t *testing.T) {
		storage := NewIntegrationSecretStorage(database.Conn(), crypto.NewNoOpEncryptor(), integration)
		err := storage.Create(core.IntegrationSecretDefinition{Value: "missing-name"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret name is required")

		err = storage.Create(core.IntegrationSecretDefinition{Name: "created-token", Value: "duplicate"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret created-token already exists")
	})

	t.Run("creates many secrets and stops on duplicates", func(t *testing.T) {
		storage := NewIntegrationSecretStorage(database.Conn(), crypto.NewNoOpEncryptor(), integration)
		err := storage.CreateMany([]core.IntegrationSecretDefinition{
			{Name: "many-one", Value: "1"},
			{Name: "many-two", Value: "2"},
		})
		require.NoError(t, err)

		err = storage.CreateMany([]core.IntegrationSecretDefinition{
			{Name: "many-three", Value: "3"},
			{Name: "many-one", Value: "duplicate"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret many-one already exists")

		value, err := storage.Get("many-three")
		require.NoError(t, err)
		assert.Equal(t, "3", value)
	})

	t.Run("updates cached and persisted secret value", func(t *testing.T) {
		storage := NewIntegrationSecretStorage(database.Conn(), crypto.NewNoOpEncryptor(), integration)
		oldValue, err := storage.Get("created-token")
		require.NoError(t, err)

		require.NoError(t, storage.Update("created-token", "updated-secret"))

		newValue, err := storage.Get("created-token")
		require.NoError(t, err)
		assert.Equal(t, "updated-secret", newValue)

		var secret models.IntegrationSecret
		err = database.Conn().
			Where("installation_id = ? AND name = ?", integration.ID, "created-token").
			First(&secret).
			Error
		require.NoError(t, err)
		assert.Equal(t, []byte("updated-secret"), secret.Value)
		assert.NotEqual(t, oldValue, string(secret.Value))

		err = storage.Update("missing", "value")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret missing not found")
	})

	t.Run("deletes cached and persisted secrets", func(t *testing.T) {
		storage := NewIntegrationSecretStorage(database.Conn(), crypto.NewNoOpEncryptor(), integration)
		require.NoError(t, storage.Delete("many-two"))

		_, err := storage.Get("many-two")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret many-two not found")

		var count int64
		err = database.Conn().
			Model(&models.IntegrationSecret{}).
			Where("installation_id = ? AND name = ?", integration.ID, "many-two").
			Count(&count).
			Error
		require.NoError(t, err)
		assert.Zero(t, count)

		err = storage.Delete("missing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret missing not found")
	})
}

func seedContextIntegrationSecret(t *testing.T, integration *models.Integration, name string, value string) {
	t.Helper()

	now := time.Now()
	secret := models.IntegrationSecret{
		OrganizationID: integration.OrganizationID,
		InstallationID: integration.ID,
		Name:           name,
		Value:          []byte(value),
		CreatedAt:      &now,
		UpdatedAt:      &now,
		Editable:       true,
	}

	require.NoError(t, database.Conn().Create(&secret).Error)
}
