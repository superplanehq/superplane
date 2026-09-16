package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__SentryAppInstallGrant(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	now := time.Now()

	grant := models.SentryAppInstallGrant{
		InstallationUUID: "install-1",
		CodeDigest:       "digest-1",
		OrganizationSlug: "acme",
		AccessToken:      []byte("sealed-access"),
		RefreshToken:     []byte("sealed-refresh"),
		TokenExpiresAt:   now.Add(time.Hour).UTC().Format(time.RFC3339),
		ExpiresAt:        now.Add(20 * time.Minute),
	}

	t.Run("a grant stays readable until it is deleted", func(t *testing.T) {
		require.NoError(t, models.UpsertSentryAppInstallGrant(db, grant))

		found, err := models.FindSentryAppInstallGrant(db, "install-1", "digest-1", now)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "acme", found.OrganizationSlug)
		assert.Equal(t, []byte("sealed-access"), found.AccessToken)
		assert.Equal(t, []byte("sealed-refresh"), found.RefreshToken)
		assert.Equal(t, grant.TokenExpiresAt, found.TokenExpiresAt)

		found, err = models.FindSentryAppInstallGrant(db, "install-1", "digest-1", now)
		require.NoError(t, err)
		require.NotNil(t, found)

		require.NoError(t, models.DeleteSentryAppInstallGrant(db, "install-1"))
		found, err = models.FindSentryAppInstallGrant(db, "install-1", "digest-1", now)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("another code cannot claim the grant", func(t *testing.T) {
		require.NoError(t, models.UpsertSentryAppInstallGrant(db, grant))

		taken, err := models.FindSentryAppInstallGrant(db, "install-1", "other-digest", now)
		require.NoError(t, err)
		assert.Nil(t, taken)

		taken, err = models.FindSentryAppInstallGrant(db, "install-1", "", now)
		require.NoError(t, err)
		assert.Nil(t, taken)
	})

	t.Run("an expired grant is not claimed", func(t *testing.T) {
		expired := grant
		expired.InstallationUUID = "install-expired"
		expired.ExpiresAt = now.Add(-time.Minute)
		require.NoError(t, models.UpsertSentryAppInstallGrant(db, expired))

		taken, err := models.FindSentryAppInstallGrant(db, "install-expired", "digest-1", now)
		require.NoError(t, err)
		assert.Nil(t, taken)
	})

	t.Run("a reinstall replaces the grant of an installation", func(t *testing.T) {
		require.NoError(t, models.UpsertSentryAppInstallGrant(db, grant))

		next := grant
		next.CodeDigest = "digest-2"
		next.AccessToken = []byte("sealed-next")
		require.NoError(t, models.UpsertSentryAppInstallGrant(db, next))

		taken, err := models.FindSentryAppInstallGrant(db, "install-1", "digest-1", now)
		require.NoError(t, err)
		assert.Nil(t, taken)

		taken, err = models.FindSentryAppInstallGrant(db, "install-1", "digest-2", now)
		require.NoError(t, err)
		require.NotNil(t, taken)
		assert.Equal(t, []byte("sealed-next"), taken.AccessToken)
	})

	t.Run("an uninstall drops the grant", func(t *testing.T) {
		require.NoError(t, models.UpsertSentryAppInstallGrant(db, grant))
		require.NoError(t, models.DeleteSentryAppInstallGrant(db, "install-1"))

		taken, err := models.FindSentryAppInstallGrant(db, "install-1", "digest-1", now)
		require.NoError(t, err)
		assert.Nil(t, taken)
	})

	t.Run("expired grants are deleted", func(t *testing.T) {
		expired := grant
		expired.InstallationUUID = "install-old"
		expired.ExpiresAt = now.Add(-time.Hour)
		require.NoError(t, models.UpsertSentryAppInstallGrant(db, expired))
		require.NoError(t, models.UpsertSentryAppInstallGrant(db, grant))
		require.NoError(t, models.DeleteExpiredSentryAppInstallGrants(db, now))

		count := int64(0)
		require.NoError(t, db.Model(&models.SentryAppInstallGrant{}).Count(&count).Error)
		assert.Equal(t, int64(1), count)
	})
}
