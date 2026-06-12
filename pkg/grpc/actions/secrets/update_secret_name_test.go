package secrets

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestAESEncryptor(t *testing.T) crypto.Encryptor {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return crypto.NewAESGCMEncryptor(key)
}

func Test__UpdateSecretName(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	encryptor := newTestAESEncryptor(t)

	t.Run("empty name -> error", func(t *testing.T) {
		_, err := UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), "any", "")
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name is required", s.Message())
	})

	t.Run("secret does not exist -> error", func(t *testing.T) {
		_, err := UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), "missing-secret", "renamed")
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "secret not found", s.Message())
	})

	t.Run("renaming a local secret re-encrypts the data with the new name", func(t *testing.T) {
		oldName := support.RandomName("secret")
		newName := support.RandomName("renamed")

		plaintext, err := json.Marshal(map[string]string{"k1": "v1", "k2": "v2"})
		require.NoError(t, err)

		encrypted, err := encryptor.Encrypt(context.Background(), plaintext, []byte(oldName))
		require.NoError(t, err)

		_, err = models.CreateSecret(oldName, secrets.ProviderLocal, uuid.NewString(), models.DomainTypeOrganization, r.Organization.ID, encrypted)
		require.NoError(t, err)

		response, err := UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), oldName, newName)
		require.NoError(t, err)
		require.NotNil(t, response.Secret)

		assert.Equal(t, newName, response.Secret.Metadata.Name)
		require.NotNil(t, response.Secret.Spec.Local)
		assert.Equal(t, map[string]string{"k1": "***", "k2": "***"}, response.Secret.Spec.Local.Data)

		stored, err := models.FindSecretByName(models.DomainTypeOrganization, r.Organization.ID, newName)
		require.NoError(t, err)

		decrypted, err := encryptor.Decrypt(context.Background(), stored.Data, []byte(newName))
		require.NoError(t, err)

		var values map[string]string
		require.NoError(t, json.Unmarshal(decrypted, &values))
		assert.Equal(t, map[string]string{"k1": "v1", "k2": "v2"}, values)
	})

	t.Run("same name is a no-op and serializes successfully", func(t *testing.T) {
		name := support.RandomName("secret")
		plaintext, err := json.Marshal(map[string]string{"only": "value"})
		require.NoError(t, err)

		encrypted, err := encryptor.Encrypt(context.Background(), plaintext, []byte(name))
		require.NoError(t, err)

		_, err = models.CreateSecret(name, secrets.ProviderLocal, uuid.NewString(), models.DomainTypeOrganization, r.Organization.ID, encrypted)
		require.NoError(t, err)

		response, err := UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), name, name)
		require.NoError(t, err)
		require.NotNil(t, response.Secret)
		assert.Equal(t, name, response.Secret.Metadata.Name)
		require.NotNil(t, response.Secret.Spec.Local)
		assert.Equal(t, map[string]string{"only": "***"}, response.Secret.Spec.Local.Data)
	})

	t.Run("rename to existing name returns InvalidArgument", func(t *testing.T) {
		first := support.RandomName("secret")
		second := support.RandomName("secret")

		firstPlaintext, err := json.Marshal(map[string]string{"a": "b"})
		require.NoError(t, err)
		firstEncrypted, err := encryptor.Encrypt(context.Background(), firstPlaintext, []byte(first))
		require.NoError(t, err)
		_, err = models.CreateSecret(first, secrets.ProviderLocal, uuid.NewString(), models.DomainTypeOrganization, r.Organization.ID, firstEncrypted)
		require.NoError(t, err)

		secondPlaintext, err := json.Marshal(map[string]string{"c": "d"})
		require.NoError(t, err)
		secondEncrypted, err := encryptor.Encrypt(context.Background(), secondPlaintext, []byte(second))
		require.NoError(t, err)
		_, err = models.CreateSecret(second, secrets.ProviderLocal, uuid.NewString(), models.DomainTypeOrganization, r.Organization.ID, secondEncrypted)
		require.NoError(t, err)

		_, err = UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), first, second)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})
}
