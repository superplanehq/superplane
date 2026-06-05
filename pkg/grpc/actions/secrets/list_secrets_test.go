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
	protos "github.com/superplanehq/superplane/pkg/protos/secrets"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListSecrets(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	encryptor := &crypto.NoOpEncryptor{}

	t.Run("no secrets", func(t *testing.T) {
		response, err := ListSecrets(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Empty(t, response.Secrets)
	})

	t.Run("secret exists", func(t *testing.T) {
		local := map[string]string{"test": "test"}
		data, _ := json.Marshal(local)

		_, err := models.CreateSecret("test", secrets.ProviderLocal, uuid.NewString(), models.DomainTypeOrganization, r.Organization.ID, data)
		require.NoError(t, err)

		response, err := ListSecrets(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Len(t, response.Secrets, 1)

		secret := response.Secrets[0]
		assert.NotEmpty(t, secret.Metadata.Id)
		assert.NotEmpty(t, secret.Metadata.CreatedAt)
		assert.Equal(t, protos.Secret_PROVIDER_LOCAL, secret.Spec.Provider)
		require.NotNil(t, secret.Spec.Local)
		require.Equal(t, map[string]string{"test": "***"}, secret.Spec.Local.Data)
	})
}

// Test__ListSecrets_DegradesGracefullyOnDecryptError ensures a single secret
// whose stored data cannot be decrypted (for example because of a historical
// rename that did not re-encrypt the payload) does not fail the whole list
// endpoint. The healthy secret should still be returned, and the corrupted
// secret should be returned with an empty key set.
func Test__ListSecrets_DegradesGracefullyOnDecryptError(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})

	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	encryptor := crypto.NewAESGCMEncryptor(key)

	healthyName := support.RandomName("healthy")
	healthyPlaintext, err := json.Marshal(map[string]string{"k": "v"})
	require.NoError(t, err)
	healthyEncrypted, err := encryptor.Encrypt(context.Background(), healthyPlaintext, []byte(healthyName))
	require.NoError(t, err)
	_, err = models.CreateSecret(healthyName, secrets.ProviderLocal, uuid.NewString(), models.DomainTypeOrganization, r.Organization.ID, healthyEncrypted)
	require.NoError(t, err)

	corruptedName := support.RandomName("corrupted")
	corruptedPlaintext, err := json.Marshal(map[string]string{"x": "y"})
	require.NoError(t, err)
	// Encrypt with a different name to simulate the "renamed without
	// re-encrypting" corruption: stored ciphertext was bound to a previous
	// name, but the row's name column has since changed.
	corruptedEncrypted, err := encryptor.Encrypt(context.Background(), corruptedPlaintext, []byte("previous-name"))
	require.NoError(t, err)
	_, err = models.CreateSecret(corruptedName, secrets.ProviderLocal, uuid.NewString(), models.DomainTypeOrganization, r.Organization.ID, corruptedEncrypted)
	require.NoError(t, err)

	response, err := ListSecrets(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Secrets, 2)

	got := map[string]map[string]string{}
	for _, s := range response.Secrets {
		require.NotNil(t, s.Spec.Local)
		got[s.Metadata.Name] = s.Spec.Local.Data
	}

	require.Contains(t, got, healthyName)
	assert.Equal(t, map[string]string{"k": "***"}, got[healthyName])

	require.Contains(t, got, corruptedName)
	assert.Empty(t, got[corruptedName], "decrypt-failed secret should serialize with empty key set")
}
