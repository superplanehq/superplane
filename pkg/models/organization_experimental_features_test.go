package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
)

func Test__ExperimentalFeatures(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("new org has empty enabled list", func(t *testing.T) {
		org, err := CreateOrganization("expfeat-new", "")
		require.NoError(t, err)

		reloaded, err := FindOrganizationByID(org.ID.String())
		require.NoError(t, err)
		assert.Empty(t, []string(reloaded.EnabledExperimentalFeatures))
	})

	t.Run("Enable adds the feature and is idempotent", func(t *testing.T) {
		org, err := CreateOrganization("expfeat-enable", "")
		require.NoError(t, err)

		require.NoError(t, EnableExperimentalFeature(org.ID, "runners"))

		reloaded, err := FindOrganizationByID(org.ID.String())
		require.NoError(t, err)
		assert.Equal(t, []string{"runners"}, []string(reloaded.EnabledExperimentalFeatures))

		require.NoError(t, EnableExperimentalFeature(org.ID, "runners"))

		reloaded, err = FindOrganizationByID(org.ID.String())
		require.NoError(t, err)
		assert.Equal(t, []string{"runners"}, []string(reloaded.EnabledExperimentalFeatures))
	})

	t.Run("Disable removes the feature and is idempotent", func(t *testing.T) {
		org, err := CreateOrganization("expfeat-disable", "")
		require.NoError(t, err)

		require.NoError(t, EnableExperimentalFeature(org.ID, "runners"))
		require.NoError(t, DisableExperimentalFeature(org.ID, "runners"))

		reloaded, err := FindOrganizationByID(org.ID.String())
		require.NoError(t, err)
		assert.Empty(t, []string(reloaded.EnabledExperimentalFeatures))

		// Disabling again is a no-op.
		require.NoError(t, DisableExperimentalFeature(org.ID, "runners"))
	})

	t.Run("Organization.HasExperimentalFeature respects enabled list", func(t *testing.T) {
		org, err := CreateOrganization("expfeat-has", "")
		require.NoError(t, err)

		assert.False(t, org.HasExperimentalFeature("runners"))

		require.NoError(t, EnableExperimentalFeature(org.ID, "runners"))
		reloaded, err := FindOrganizationByID(org.ID.String())
		require.NoError(t, err)
		assert.True(t, reloaded.HasExperimentalFeature("runners"))
	})

	t.Run("released features short-circuit to true regardless of stored list", func(t *testing.T) {
		original := stubReleasedFeature(t, "graduated")
		t.Cleanup(original)

		org, err := CreateOrganization("expfeat-released", "")
		require.NoError(t, err)

		assert.True(t, org.HasExperimentalFeature("graduated"))

		ok, err := HasExperimentalFeature(org.ID, "graduated")
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("HasExperimentalFeature loads org when feature is not released", func(t *testing.T) {
		org, err := CreateOrganization("expfeat-load", "")
		require.NoError(t, err)

		ok, err := HasExperimentalFeature(org.ID, "runners")
		require.NoError(t, err)
		assert.False(t, ok)

		require.NoError(t, EnableExperimentalFeature(org.ID, "runners"))

		ok, err = HasExperimentalFeature(org.ID, "runners")
		require.NoError(t, err)
		assert.True(t, ok)
	})
}

// stubReleasedFeature monkey-patches the registry to include a released
// feature with the given id. Returns a cleanup function that restores the
// original registry.
func stubReleasedFeature(t *testing.T, id string) func() {
	t.Helper()
	released := true
	return features.WithRegistryForTest([]features.Feature{{ID: id, Label: id, Released: &released}})
}
