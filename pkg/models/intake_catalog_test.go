package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__IntakeCatalog(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	t.Run("creation follows the feature flag and status", func(t *testing.T) {
		other, err := models.CreateOrganization(support.RandomName("org"), support.RandomName("org"))
		require.NoError(t, err)
		require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureFactoryDatadogIntake))
		t.Cleanup(func() {
			require.NoError(t, models.DisableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureFactoryDatadogIntake))
		})

		cases := []struct {
			name      string
			status    string
			wantAdded bool
			wantOther bool
		}{
			{name: "planned", status: models.IntakeStatusPlanned, wantAdded: false, wantOther: false},
			{name: "internal", status: models.IntakeStatusAlpha, wantAdded: true, wantOther: false},
			{name: "beta", status: models.IntakeStatusBeta, wantAdded: true, wantOther: false},
			{name: "generally available status", status: models.IntakeStatusGA, wantAdded: true, wantOther: false},
			{name: "deprecated", status: models.IntakeStatusDeprecated, wantAdded: false, wantOther: false},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				entry := withCatalogEntryState(t, db, models.FactoryIntakeSourceDatadog, tc.status)
				available, err := models.IsIntakeAvailableForOrganization(db, entry.Key, r.Organization.ID)
				require.NoError(t, err)
				assert.Equal(t, tc.wantAdded, available)

				available, err = models.IsIntakeAvailableForOrganization(db, entry.Key, other.ID)
				require.NoError(t, err)
				assert.Equal(t, tc.wantOther, available)
			})
		}
	})

	t.Run("an intake with no feature flag is visible to every company", func(t *testing.T) {
		available, err := models.IsIntakeAvailableForOrganization(db, models.FactoryIntakeSourceGitHubIssues, r.Organization.ID)
		require.NoError(t, err)
		assert.True(t, available)
	})

	t.Run("an unknown key is not available", func(t *testing.T) {
		available, err := models.IsIntakeAvailableForOrganization(db, "does-not-exist", r.Organization.ID)
		require.NoError(t, err)
		assert.False(t, available)
	})

	t.Run("an unimplemented intake stays planned", func(t *testing.T) {
		entry := createPlannedEntry(t, db, "acme-tracker")

		status := models.IntakeStatusBeta
		err := entry.Update(db, models.IntakeCatalogPatch{Status: &status}, nil)
		assert.ErrorIs(t, err, models.ErrIntakeCatalogNotImplemented)
		stored, err := models.FindIntakeCatalogEntry(db, entry.Key)
		require.NoError(t, err)
		assert.Equal(t, models.IntakeStatusPlanned, stored.Status)
	})

	t.Run("an admin can set the status without changing who can see the intake", func(t *testing.T) {
		entry := withCatalogEntryState(t, db, models.FactoryIntakeSourceDatadog, models.IntakeStatusBeta)

		status := models.IntakeStatusAlpha
		require.NoError(t, entry.Update(db, models.IntakeCatalogPatch{Status: &status}, &r.Account.ID))

		stored, err := models.FindIntakeCatalogEntry(db, entry.Key)
		require.NoError(t, err)
		assert.Equal(t, models.IntakeStatusAlpha, stored.Status)
		assert.Equal(t, r.Account.Name, stored.UpdatedByName)
		assert.False(t, stored.VisibleTo(false))
		assert.True(t, stored.VisibleTo(true))
	})

	t.Run("creating a duplicate key fails", func(t *testing.T) {
		entry, err := models.NewIntakeCatalogEntry(models.FactoryIntakeSourceGitHubIssues, "Copy", models.IntakeCategoryIssueTracking, "")
		require.NoError(t, err)
		assert.ErrorIs(t, models.CreateIntakeCatalogEntry(db, entry, nil), models.ErrIntakeCatalogEntryExists)
	})

	t.Run("rejects keys that are not slugs", func(t *testing.T) {
		_, err := models.NewIntakeCatalogEntry("Acme Tracker", "Acme", models.IntakeCategoryIssueTracking, "")
		assert.ErrorIs(t, err, models.ErrIntakeCatalogKeyInvalid)
	})

	t.Run("only unimplemented planned intakes can be deleted", func(t *testing.T) {
		implemented, err := models.FindIntakeCatalogEntry(db, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)
		assert.ErrorIs(t, implemented.Delete(db), models.ErrIntakeCatalogDeleteImplemented)

		planned := createPlannedEntry(t, db, "acme-delete")
		require.NoError(t, planned.Delete(db))
		_, err = models.FindIntakeCatalogEntry(db, planned.Key)
		assert.ErrorIs(t, err, models.ErrIntakeCatalogEntryNotFound)
	})
}

// withCatalogEntryState sets a seeded entry to a state for one test and
// restores the seeded state afterwards, because the seed rows outlive tests.
func withCatalogEntryState(t *testing.T, db *gorm.DB, key, status string) *models.IntakeCatalogEntry {
	t.Helper()

	entry, err := models.FindIntakeCatalogEntry(db, key)
	require.NoError(t, err)
	original := *entry
	t.Cleanup(func() {
		require.NoError(t, db.Model(&models.IntakeCatalogEntry{}).Where("key = ?", key).Update("status", original.Status).Error)
	})

	require.NoError(t, db.Model(&models.IntakeCatalogEntry{}).Where("key = ?", key).Update("status", status).Error)

	entry, err = models.FindIntakeCatalogEntry(db, key)
	require.NoError(t, err)
	return entry
}

func createPlannedEntry(t *testing.T, db *gorm.DB, key string) *models.IntakeCatalogEntry {
	t.Helper()

	entry, err := models.NewIntakeCatalogEntry(key, "Acme", models.IntakeCategoryIssueTracking, "")
	require.NoError(t, err)
	require.NoError(t, models.CreateIntakeCatalogEntry(db, entry, nil))
	t.Cleanup(func() {
		require.NoError(t, db.Where("key = ?", key).Delete(&models.IntakeCatalogEntry{}).Error)
	})
	return entry
}
