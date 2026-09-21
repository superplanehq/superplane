package secrets

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
)

func testEncryptor(t *testing.T) crypto.Encryptor {
	t.Helper()

	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	return crypto.NewAESGCMEncryptor(key)
}

func nameBoundSecret(t *testing.T, encryptor crypto.Encryptor, name string, values map[string]string) *models.Secret {
	t.Helper()

	raw, err := json.Marshal(values)
	require.NoError(t, err)

	data, err := encryptor.Encrypt(context.Background(), raw, []byte(name))
	require.NoError(t, err)

	return &models.Secret{ID: uuid.New(), Name: name, Data: data}
}

func Test__LocalCrypto(t *testing.T) {
	ctx := context.Background()
	values := map[string]string{"username": "admin", "password": "s3cret"}

	t.Run("payload written under the ID reads back", func(t *testing.T) {
		encryptor := testEncryptor(t)
		secret := &models.Secret{ID: uuid.New(), Name: "creds"}

		data, err := EncryptLocalData(ctx, encryptor, secret.ID, values)
		require.NoError(t, err)
		secret.Data = data

		got, err := DecryptLocalData(ctx, encryptor, secret)
		require.NoError(t, err)
		require.Equal(t, values, got)
	})

	t.Run("renaming does not require re-encryption", func(t *testing.T) {
		encryptor := testEncryptor(t)
		secret := &models.Secret{ID: uuid.New(), Name: "creds"}

		data, err := EncryptLocalData(ctx, encryptor, secret.ID, values)
		require.NoError(t, err)
		secret.Data = data

		secret.Name = "renamed-creds"

		got, err := DecryptLocalData(ctx, encryptor, secret)
		require.NoError(t, err)
		require.Equal(t, values, got)
	})

	t.Run("payload written under the name still reads back", func(t *testing.T) {
		encryptor := testEncryptor(t)
		secret := nameBoundSecret(t, encryptor, "legacy-creds", values)

		got, err := DecryptLocalData(ctx, encryptor, secret)
		require.NoError(t, err)
		require.Equal(t, values, got)
	})

	t.Run("payload bound to neither the ID nor the name fails", func(t *testing.T) {
		encryptor := testEncryptor(t)
		secret := nameBoundSecret(t, encryptor, "legacy-creds", values)
		secret.Name = "some-other-name"

		_, err := DecryptLocalData(ctx, encryptor, secret)
		require.Error(t, err)
	})

	t.Run("empty payload reads back as an empty map", func(t *testing.T) {
		got, err := DecryptLocalData(ctx, crypto.NewNoOpEncryptor(), &models.Secret{ID: uuid.New(), Name: "empty"})
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("rebinding leaves an ID-bound payload alone", func(t *testing.T) {
		encryptor := testEncryptor(t)
		secret := &models.Secret{ID: uuid.New(), Name: "creds"}

		data, err := EncryptLocalData(ctx, encryptor, secret.ID, values)
		require.NoError(t, err)
		secret.Data = data

		rebound, err := RebindLocalData(ctx, encryptor, secret)
		require.NoError(t, err)
		require.Nil(t, rebound)
	})

	t.Run("rebinding a name-bound payload survives the rename", func(t *testing.T) {
		encryptor := testEncryptor(t)
		secret := nameBoundSecret(t, encryptor, "legacy-creds", values)

		rebound, err := RebindLocalData(ctx, encryptor, secret)
		require.NoError(t, err)
		require.NotNil(t, rebound)

		secret.Data = rebound
		secret.Name = "renamed-creds"

		got, err := DecryptLocalData(ctx, encryptor, secret)
		require.NoError(t, err)
		require.Equal(t, values, got)
	})

	t.Run("rebinding a payload bound to neither fails", func(t *testing.T) {
		encryptor := testEncryptor(t)
		secret := nameBoundSecret(t, encryptor, "legacy-creds", values)
		secret.Name = "some-other-name"

		_, err := RebindLocalData(ctx, encryptor, secret)
		require.Error(t, err)
	})
}
