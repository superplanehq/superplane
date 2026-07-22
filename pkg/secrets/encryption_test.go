package secrets

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestLocalDataEncryption(t *testing.T) {
	encryptor := crypto.NewAESGCMEncryptor([]byte("1234567890abcdefghijklmnopqrstuv"))
	data := map[string]string{"token": "secret"}

	t.Run("uses secret ID as associated data", func(t *testing.T) {
		secret := models.Secret{ID: uuid.New(), Name: "renamable-secret"}

		encrypted, err := EncryptLocalData(context.Background(), encryptor, secret, data)
		require.NoError(t, err)
		secret.Data = encrypted

		decrypted, err := DecryptLocalData(context.Background(), encryptor, secret)
		require.NoError(t, err)
		require.Equal(t, data, decrypted)

		_, err = encryptor.Decrypt(context.Background(), encrypted, []byte(secret.Name))
		require.Error(t, err)
	})

	t.Run("falls back to legacy name associated data", func(t *testing.T) {
		secret := models.Secret{ID: uuid.New(), Name: "legacy-secret"}
		raw, err := json.Marshal(data)
		require.NoError(t, err)

		encrypted, err := encryptor.Encrypt(context.Background(), raw, []byte(secret.Name))
		require.NoError(t, err)
		secret.Data = encrypted

		decrypted, err := DecryptLocalData(context.Background(), encryptor, secret)
		require.NoError(t, err)
		require.Equal(t, data, decrypted)
	})

	t.Run("returns both ID and legacy decrypt failures", func(t *testing.T) {
		secret := models.Secret{ID: uuid.New(), Name: "broken-secret"}
		raw, err := json.Marshal(data)
		require.NoError(t, err)

		encrypted, err := encryptor.Encrypt(context.Background(), raw, []byte("wrong-associated-data"))
		require.NoError(t, err)
		secret.Data = encrypted

		_, err = DecryptLocalDataRaw(context.Background(), encryptor, secret)
		require.Error(t, err)
		require.Contains(t, err.Error(), "secret ID associated data")
		require.Contains(t, err.Error(), "legacy name associated data")
	})
}
