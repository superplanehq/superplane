package licensing_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestDatabaseSource(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	ctx := context.Background()
	encryptor := crypto.NewAESGCMEncryptor([]byte("01234567890123456789012345678901"))
	source := licensing.NewDatabaseSource(encryptor)
	issuer := licensingtest.NewIssuer("test-2026-01")
	account, err := models.CreateAccount("License Owner", "license-owner@example.com")
	require.NoError(t, err)

	t.Run("reports no license before install", func(t *testing.T) {
		_, err := source.Read(ctx)
		require.ErrorIs(t, err, licensing.ErrNotInstalled)
	})

	t.Run("stores the license encrypted at rest", func(t *testing.T) {
		token := issuer.License(licensing.FeatureGroups)
		require.NoError(t, source.Write(ctx, token, account.ID))

		record, err := models.FindInstallationLicense(database.Conn())
		require.NoError(t, err)
		assert.False(t, bytes.Contains(record.EncryptedLicense, token))
		require.NotNil(t, record.InstalledBy)
		assert.Equal(t, account.ID, *record.InstalledBy)

		raw, err := source.Read(ctx)
		require.NoError(t, err)
		assert.Equal(t, token, raw)
	})

	t.Run("replaces the existing license", func(t *testing.T) {
		replacement := issuer.License(licensing.FeatureCustomRoles)
		require.NoError(t, source.Write(ctx, replacement, account.ID))

		raw, err := source.Read(ctx)
		require.NoError(t, err)
		assert.Equal(t, replacement, raw)
	})

	t.Run("rejects ciphertext encrypted for another purpose", func(t *testing.T) {
		token := issuer.License(licensing.FeatureGroups)
		encrypted, err := encryptor.Encrypt(ctx, token, []byte("some-secret-name"))
		require.NoError(t, err)
		require.NoError(t, models.SaveInstallationLicense(database.Conn(), encrypted, nil))

		_, err = source.Read(ctx)
		assert.Equal(t, licensing.ReasonUnreadable, licensing.ReasonOf(err))
	})

	t.Run("clear removes the license", func(t *testing.T) {
		require.NoError(t, source.Clear(ctx))

		_, err := source.Read(ctx)
		require.ErrorIs(t, err, licensing.ErrNotInstalled)
	})
}

func TestDatabaseKeyListCache(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	ctx := context.Background()
	cache := licensing.DatabaseKeyListCache{}
	root := licensingtest.NewIssuer("root-2026")
	signer := licensingtest.NewIssuer("signing-2026-01")
	verified := func(version int64) *licensing.KeyList {
		list, err := licensing.VerifyKeyList(root.KeyList(version, signer), licensingtest.KeySet(root))
		require.NoError(t, err)
		return list
	}

	document, err := cache.Load(ctx)
	require.NoError(t, err)
	assert.Empty(t, document)

	newer := verified(3)
	require.NoError(t, cache.Save(ctx, newer))
	require.NoError(t, cache.Save(ctx, verified(2)))

	document, err = cache.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, newer.Document, document, "an older list never replaces a newer one")
}
