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

func Test__UpdateSecretName(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})

	key := make([]byte, 32)
	_, _ = rand.Read(key)
	encryptor := crypto.NewAESGCMEncryptor(key)

	plain := map[string]string{"k": "v"}
	rawData, _ := json.Marshal(plain)

	encryptedData, err := encryptor.Encrypt(context.Background(), rawData, []byte("original"))
	require.NoError(t, err)

	secret, err := models.CreateSecret(
		"original",
		secrets.ProviderLocal,
		uuid.NewString(),
		models.DomainTypeOrganization,
		r.Organization.ID,
		encryptedData,
	)
	require.NoError(t, err)

	t.Run("name is required", func(t *testing.T) {
		_, err := UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), secret.ID.String(), "")
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("secret does not exist -> error", func(t *testing.T) {
		_, err := UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), "nope", "new-name")
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "secret not found", s.Message())
	})

	t.Run("same name is a no-op and still serializes the secret", func(t *testing.T) {
		response, err := UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), secret.ID.String(), "original")
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Secret)
		assert.Equal(t, "original", response.Secret.Metadata.Name)
	})

	t.Run("rename re-encrypts data with the new name and serializes successfully", func(t *testing.T) {
		response, err := UpdateSecretName(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), secret.ID.String(), "renamed")
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Secret)
		assert.Equal(t, "renamed", response.Secret.Metadata.Name)
		require.NotNil(t, response.Secret.Spec.Local)
		assert.Equal(t, map[string]string{"k": "***"}, response.Secret.Spec.Local.Data)

		stored, err := models.FindSecretByID(models.DomainTypeOrganization, r.Organization.ID, secret.ID.String())
		require.NoError(t, err)
		decrypted, err := encryptor.Decrypt(context.Background(), stored.Data, []byte("renamed"))
		require.NoError(t, err)
		var got map[string]string
		require.NoError(t, json.Unmarshal(decrypted, &got))
		assert.Equal(t, plain, got)
	})
}
