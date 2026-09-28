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

func Test__HostedAppInstallation(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	now := time.Now().UTC().Truncate(time.Second)
	provider := models.HostedAppProviderGitHub

	row := models.HostedAppInstallation{
		Provider:       provider,
		InstallationID: "11",
		AccountLogin:   "acme",
		AccountType:    "Organization",
		AccountID:      99,
		SenderLogin:    "member",
		LastEventAt:    now,
	}

	t.Run("upsert then find", func(t *testing.T) {
		require.NoError(t, models.UpsertHostedAppInstallation(db, row))

		found, err := models.FindHostedAppInstallation(db, provider, "11")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, provider, found.Provider)
		assert.Equal(t, "acme", found.AccountLogin)
		assert.Equal(t, "Organization", found.AccountType)
		assert.Equal(t, int64(99), found.AccountID)
		assert.Equal(t, "member", found.SenderLogin)
		assert.False(t, found.DeletedAt.Valid)
	})

	t.Run("find by account login is case insensitive", func(t *testing.T) {
		found, err := models.FindHostedAppInstallationByAccountLogin(db, provider, "Acme")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "11", found.InstallationID)
	})

	t.Run("find is scoped to the provider", func(t *testing.T) {
		found, err := models.FindHostedAppInstallation(db, "sentry", "11")
		require.NoError(t, err)
		assert.Nil(t, found)

		byAccount, err := models.FindHostedAppInstallationByAccountLogin(db, "sentry", "acme")
		require.NoError(t, err)
		assert.Nil(t, byAccount)
	})

	t.Run("touch updates last event time", func(t *testing.T) {
		later := now.Add(2 * time.Minute)
		require.NoError(t, models.TouchHostedAppInstallation(db, provider, "11", later))

		found, err := models.FindHostedAppInstallation(db, provider, "11")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.True(t, found.LastEventAt.Equal(later) || found.LastEventAt.After(later.Add(-time.Second)))
	})

	t.Run("soft delete hides the account lookup and keeps the id lookup", func(t *testing.T) {
		require.NoError(t, models.SoftDeleteHostedAppInstallation(db, provider, "11"))

		found, err := models.FindHostedAppInstallation(db, provider, "11")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.True(t, found.DeletedAt.Valid)

		byAccount, err := models.FindHostedAppInstallationByAccountLogin(db, provider, "acme")
		require.NoError(t, err)
		assert.Nil(t, byAccount)
	})

	t.Run("upsert after delete restores the row", func(t *testing.T) {
		next := row
		next.SenderLogin = "owner"
		next.LastEventAt = now.Add(time.Minute)
		require.NoError(t, models.UpsertHostedAppInstallation(db, next))

		found, err := models.FindHostedAppInstallation(db, provider, "11")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.False(t, found.DeletedAt.Valid)
		assert.Equal(t, "owner", found.SenderLogin)
	})

	t.Run("empty installation id is a no-op", func(t *testing.T) {
		require.NoError(t, models.UpsertHostedAppInstallation(db, models.HostedAppInstallation{Provider: provider}))
		found, err := models.FindHostedAppInstallation(db, provider, "")
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("reconcile keeps last_event_at and restores a deleted row", func(t *testing.T) {
		stale := now.Add(-2 * time.Hour)
		require.NoError(t, models.UpsertHostedAppInstallation(db, models.HostedAppInstallation{
			Provider:       provider,
			InstallationID: "21",
			AccountLogin:   "old-org",
			AccountType:    "Organization",
			LastEventAt:    stale,
		}))
		require.NoError(t, models.SoftDeleteHostedAppInstallation(db, provider, "21"))

		require.NoError(t, models.ReconcileHostedAppInstallation(db, models.HostedAppInstallation{
			Provider:       provider,
			InstallationID: "21",
			AccountLogin:   "old-org-renamed",
			AccountType:    "Organization",
			LastEventAt:    now,
		}))

		found, err := models.FindHostedAppInstallation(db, provider, "21")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "old-org-renamed", found.AccountLogin)
		assert.False(t, found.DeletedAt.Valid)
		assert.True(t, found.LastEventAt.Before(now.Add(-time.Hour)), "reconcile must not freshen last_event_at")
	})

	t.Run("reconcile inserts a new row with its provided time", func(t *testing.T) {
		createdAt := now.Add(-30 * time.Minute)
		require.NoError(t, models.ReconcileHostedAppInstallation(db, models.HostedAppInstallation{
			Provider:       provider,
			InstallationID: "22",
			AccountLogin:   "fresh-org",
			AccountType:    "Organization",
			LastEventAt:    createdAt,
		}))

		found, err := models.FindHostedAppInstallation(db, provider, "22")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.True(t, found.LastEventAt.Before(now.Add(-29*time.Minute)))
	})

	t.Run("reconcile without timestamps stays outside the claim window", func(t *testing.T) {
		require.NoError(t, models.ReconcileHostedAppInstallation(db, models.HostedAppInstallation{
			Provider:       provider,
			InstallationID: "23",
			AccountLogin:   "timeless-org",
			AccountType:    "Organization",
		}))

		found, err := models.FindHostedAppInstallation(db, provider, "23")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.True(t, found.LastEventAt.Before(now.Add(-24*time.Hour)),
			"a missing provider timestamp must not look fresh")
	})

	t.Run("list returns live rows only", func(t *testing.T) {
		require.NoError(t, models.SoftDeleteHostedAppInstallation(db, provider, "22"))

		rows, err := models.ListHostedAppInstallations(db, provider)
		require.NoError(t, err)
		logins := make([]string, 0, len(rows))
		for _, row := range rows {
			logins = append(logins, row.AccountLogin)
		}
		assert.Contains(t, logins, "old-org-renamed")
		assert.NotContains(t, logins, "fresh-org")
	})
}
