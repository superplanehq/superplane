package encryption

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type cannotDecryptPrimary struct {
	crypto.MasterKeys
	primary string
}

func (m cannotDecryptPrimary) Unwrap(ctx context.Context, reference string, wrapped, associatedData []byte) ([]byte, error) {
	if reference == m.primary {
		return nil, errors.New("KMS decrypt denied")
	}
	return m.MasterKeys.Unwrap(ctx, reference, wrapped, associatedData)
}

func storeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := database.DB(context.Background())
	require.NoError(t, database.VerifyTestDatabase(db))
	clean := func() {
		require.NoError(t, db.Exec("DELETE FROM application_encryption_migrations").Error)
		require.NoError(t, db.Exec("DELETE FROM application_encryption_keys").Error)
	}
	clean()
	t.Cleanup(clean)
	return db
}

func storeTestMasterKeys(t *testing.T) crypto.MasterKeys {
	t.Helper()
	v1 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	v2 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32))
	masters, err := crypto.ParseLocalMasterKeys("local-v1:" + v1 + ",local-v2:" + v2)
	require.NoError(t, err)
	return masters
}

func TestStoreCreatesDataKeysAndRewrapsMaster(t *testing.T) {
	ctx := t.Context()
	db := storeTestDB(t)
	masters := storeTestMasterKeys(t)
	v2 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32))
	store := NewStore(masters, "local-v1")
	require.NoError(t, store.Initialize(ctx))
	encryptor := crypto.NewVersionedEncryptor(store, nil)
	oldValue, err := encryptor.Encrypt(ctx, []byte("old value"), []byte("record"))
	require.NoError(t, err)

	store = NewStore(masters, "local-v2")
	require.NoError(t, store.Initialize(ctx))
	encryptor = crypto.NewVersionedEncryptor(store, nil)
	newID, err := store.CreateAndActivateDataKey(ctx)
	require.NoError(t, err)
	newValue, err := encryptor.Encrypt(ctx, []byte("new value"), []byte("record"))
	require.NoError(t, err)
	keys, activeID, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, keys, 2)
	require.Equal(t, newID, activeID)
	require.Equal(t, models.ApplicationEncryptionKeyActive, keys[1].State)
	newKey, err := models.LoadApplicationEncryptionKey(db, newID)
	require.NoError(t, err)
	require.Equal(t, "local-v2", newKey.MasterKeyRef)
	require.NoError(t, store.Rewrap(ctx))

	for _, item := range []struct {
		ciphertext []byte
		plain      string
	}{{oldValue, "old value"}, {newValue, "new value"}} {
		decrypted, err := encryptor.Decrypt(ctx, item.ciphertext, []byte("record"))
		require.NoError(t, err)
		require.Equal(t, item.plain, string(decrypted))
	}

	v2Only, err := crypto.ParseLocalMasterKeys("local-v2:" + v2)
	require.NoError(t, err)
	require.NoError(t, NewStore(v2Only, "local-v2").Initialize(ctx))
	for _, key := range keys {
		stored, err := models.LoadApplicationEncryptionKey(db, key.ID)
		require.NoError(t, err)
		require.Equal(t, "local-v2", stored.MasterKeyRef)
	}
}

func TestStoreEncryptsWithActiveKeyInsideTransaction(t *testing.T) {
	ctx := t.Context()
	db := storeTestDB(t)
	store := NewStore(storeTestMasterKeys(t), "local-v1")
	require.NoError(t, store.Initialize(ctx))
	encryptor := NewApplicationEncryptor(crypto.NewVersionedEncryptor(store, nil), store)

	var ciphertext []byte
	var newID uuid.UUID
	err := db.Transaction(func(tx *gorm.DB) error {
		active, err := models.LoadActiveApplicationEncryptionKey(tx)
		if err != nil {
			return err
		}
		key, err := store.createKey(ctx, "local-v1")
		if err != nil {
			return err
		}
		if err := key.CreateAndActivate(tx, active); err != nil {
			return err
		}
		newID = key.ID
		ciphertext, err = ForDB(tx, encryptor).Encrypt(ctx, []byte("secret"), []byte("record"))
		return err
	})
	require.NoError(t, err)
	keyID, versioned, err := crypto.EncryptedDataKeyID(ciphertext)
	require.NoError(t, err)
	require.True(t, versioned)
	require.Equal(t, newID, keyID)
}

func TestStoreRewrapKeepsOriginalKeysWhenPrimaryCannotDecrypt(t *testing.T) {
	ctx := t.Context()
	db := storeTestDB(t)
	masters := storeTestMasterKeys(t)
	store := NewStore(masters, "local-v1")
	require.NoError(t, store.Initialize(ctx))
	before, err := models.ListApplicationEncryptionKeys(db)
	require.NoError(t, err)

	store = NewStore(cannotDecryptPrimary{MasterKeys: masters, primary: "local-v2"}, "local-v2")
	require.ErrorContains(t, store.Rewrap(ctx), "KMS decrypt denied")
	after, err := models.ListApplicationEncryptionKeys(db)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestStoreCreateKeepsActiveKeyWhenPrimaryCannotDecrypt(t *testing.T) {
	ctx := t.Context()
	db := storeTestDB(t)
	masters := storeTestMasterKeys(t)
	store := NewStore(masters, "local-v1")
	require.NoError(t, store.Initialize(ctx))
	before, err := models.ListApplicationEncryptionKeys(db)
	require.NoError(t, err)

	store = NewStore(cannotDecryptPrimary{MasterKeys: masters, primary: "local-v2"}, "local-v2")
	_, err = store.CreateAndActivateDataKey(ctx)
	require.ErrorContains(t, err, "KMS decrypt denied")
	after, err := models.ListApplicationEncryptionKeys(db)
	require.NoError(t, err)
	require.Equal(t, before, after)
}
