package secrets

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__UpdateSecretName(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	encryptor := crypto.NewAESGCMEncryptor([]byte("1234567890abcdefghijklmnopqrstuv"))

	createNameBoundSecret := func(t *testing.T, name string, data map[string]string) *models.Secret {
		t.Helper()
		raw, err := json.Marshal(data)
		require.NoError(t, err)
		encrypted, err := encryptor.Encrypt(context.Background(), raw, []byte(name))
		require.NoError(t, err)
		secret, err := models.CreateSecret(uuid.New(), name, secrets.ProviderLocal, r.User.String(), models.DomainTypeOrganization, r.Organization.ID, encrypted)
		require.NoError(t, err)
		return secret
	}

	createIDBoundSecret := func(t *testing.T, name string, data map[string]string) *models.Secret {
		t.Helper()
		secretID := uuid.New()
		encrypted, err := secrets.EncryptLocalData(context.Background(), encryptor, secretID, data)
		require.NoError(t, err)
		secret, err := models.CreateSecret(secretID, name, secrets.ProviderLocal, r.User.String(), models.DomainTypeOrganization, r.Organization.ID, encrypted)
		require.NoError(t, err)
		return secret
	}

	t.Run("renaming an ID-bound secret leaves the stored payload untouched", func(t *testing.T) {
		oldName := support.RandomName("secret")
		newName := support.RandomName("secret-renamed")
		plainData := map[string]string{"key": "value"}
		before := createIDBoundSecret(t, oldName, plainData)

		_, err := UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), oldName, newName)
		require.NoError(t, err)

		after, err := models.FindSecretByName(models.DomainTypeOrganization, r.Organization.ID, newName)
		require.NoError(t, err)
		assert.Equal(t, before.Data, after.Data)

		decrypted, err := secrets.DecryptLocalData(context.Background(), encryptor, after)
		require.NoError(t, err)
		assert.Equal(t, plainData, decrypted)
	})

	t.Run("renaming a name-bound secret rebinds the payload to the ID", func(t *testing.T) {
		oldName := support.RandomName("secret")
		newName := support.RandomName("secret-renamed")
		plainData := map[string]string{"key": "value"}
		before := createNameBoundSecret(t, oldName, plainData)

		_, err := UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), oldName, newName)
		require.NoError(t, err)

		secret, err := models.FindSecretByName(models.DomainTypeOrganization, r.Organization.ID, newName)
		require.NoError(t, err)
		assert.Equal(t, newName, secret.Name)
		assert.NotEqual(t, before.Data, secret.Data)

		decrypted, err := secrets.DecryptLocalData(context.Background(), encryptor, secret)
		require.NoError(t, err)
		assert.Equal(t, plainData, decrypted)
	})

	t.Run("key operations work after rename", func(t *testing.T) {
		oldName := support.RandomName("secret")
		newName := support.RandomName("secret-renamed")
		plainData := map[string]string{"key": "value", "key2": "value2"}
		createNameBoundSecret(t, oldName, plainData)

		_, err := UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), oldName, newName)
		require.NoError(t, err)

		_, err = DeleteSecretKey(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), newName, "key2")
		require.NoError(t, err)

		secret, err := models.FindSecretByName(models.DomainTypeOrganization, r.Organization.ID, newName)
		require.NoError(t, err)

		decrypted, err := secrets.DecryptLocalData(context.Background(), encryptor, secret)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"key": "value"}, decrypted)
	})

	t.Run("same name is a no-op", func(t *testing.T) {
		name := support.RandomName("secret")
		plainData := map[string]string{"key": "value"}
		createNameBoundSecret(t, name, plainData)

		_, err := UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), name, name)
		require.NoError(t, err)

		secret, err := models.FindSecretByName(models.DomainTypeOrganization, r.Organization.ID, name)
		require.NoError(t, err)

		decrypted, err := secrets.DecryptLocalData(context.Background(), encryptor, secret)
		require.NoError(t, err)
		assert.Equal(t, plainData, decrypted)
	})

	t.Run("name already used", func(t *testing.T) {
		existingName := support.RandomName("secret")
		otherName := support.RandomName("secret")
		existingData := map[string]string{"key": "existing"}
		otherData := map[string]string{"key": "other"}
		createNameBoundSecret(t, existingName, existingData)
		createNameBoundSecret(t, otherName, otherData)

		secretBefore, err := models.FindSecretByName(models.DomainTypeOrganization, r.Organization.ID, otherName)
		require.NoError(t, err)

		_, err = UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), otherName, existingName)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "invalid secret name", msg)

		secretAfter, err := models.FindSecretByName(models.DomainTypeOrganization, r.Organization.ID, otherName)
		require.NoError(t, err)
		assert.Equal(t, secretBefore.Name, secretAfter.Name)
		assert.Equal(t, secretBefore.Data, secretAfter.Data)

		decrypted, err := secrets.DecryptLocalData(context.Background(), encryptor, secretAfter)
		require.NoError(t, err)
		assert.Equal(t, otherData, decrypted)
	})
}
