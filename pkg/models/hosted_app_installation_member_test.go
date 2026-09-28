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

func Test__HostedAppInstallationMember(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	now := time.Now().UTC().Truncate(time.Second)
	provider := models.HostedAppProviderGitHub

	t.Run("upsert then find is case insensitive", func(t *testing.T) {
		require.NoError(t, models.UpsertHostedAppInstallationMember(db, models.HostedAppInstallationMember{
			Provider:       provider,
			InstallationID: "11",
			MemberLogin:    "Member-One",
			Allowed:        true,
			CheckedAt:      now,
		}))

		found, err := models.FindHostedAppInstallationMember(db, provider, "11", "member-one")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "member-one", found.MemberLogin)
		assert.True(t, found.Allowed)

		byMixedCase, err := models.FindHostedAppInstallationMember(db, provider, "11", "MEMBER-ONE")
		require.NoError(t, err)
		require.NotNil(t, byMixedCase)
	})

	t.Run("upsert replaces the previous check result", func(t *testing.T) {
		later := now.Add(time.Minute)
		require.NoError(t, models.UpsertHostedAppInstallationMember(db, models.HostedAppInstallationMember{
			Provider:       provider,
			InstallationID: "11",
			MemberLogin:    "member-one",
			Allowed:        false,
			Errored:        true,
			CheckedAt:      later,
		}))

		found, err := models.FindHostedAppInstallationMember(db, provider, "11", "member-one")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.False(t, found.Allowed)
		assert.True(t, found.Errored)
		assert.True(t, found.CheckedAt.After(now.Add(-time.Second)))
	})

	t.Run("find is scoped to provider and installation", func(t *testing.T) {
		found, err := models.FindHostedAppInstallationMember(db, "sentry", "11", "member-one")
		require.NoError(t, err)
		assert.Nil(t, found)

		found, err = models.FindHostedAppInstallationMember(db, provider, "12", "member-one")
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("empty login is a no-op", func(t *testing.T) {
		require.NoError(t, models.UpsertHostedAppInstallationMember(db, models.HostedAppInstallationMember{
			Provider:       provider,
			InstallationID: "11",
		}))
		found, err := models.FindHostedAppInstallationMember(db, provider, "11", "")
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}
