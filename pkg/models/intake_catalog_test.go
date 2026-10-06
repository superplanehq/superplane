package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__IntakeCatalog(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	t.Run("availability follows status, open beta, and added companies", func(t *testing.T) {
		other, err := models.CreateOrganization(support.RandomName("org"), support.RandomName("org"))
		require.NoError(t, err)

		cases := []struct {
			name          string
			status        string
			enabledForAll bool
			addOrg        bool
			wantAdded     bool
			wantOther     bool
		}{
			{name: "planned", status: models.IntakeStatusPlanned, addOrg: true, wantAdded: false, wantOther: false},
			{name: "internal", status: models.IntakeStatusAlpha, addOrg: true, wantAdded: true, wantOther: false},
			{name: "closed beta", status: models.IntakeStatusBeta, addOrg: true, wantAdded: true, wantOther: false},
			{name: "open beta", status: models.IntakeStatusBeta, enabledForAll: true, wantAdded: true, wantOther: true},
			{name: "generally available", status: models.IntakeStatusGA, wantAdded: true, wantOther: true},
			{name: "deprecated", status: models.IntakeStatusDeprecated, addOrg: true, wantAdded: false, wantOther: false},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				entry := withCatalogEntryState(t, db, models.FactoryIntakeSourceDatadog, tc.status, tc.enabledForAll)
				require.NoError(t, entry.RemoveOrganization(db, r.Organization.ID))
				if tc.addOrg || tc.wantAdded {
					require.NoError(t, entry.AddOrganization(db, r.Organization.ID))
				}

				available, err := models.IsIntakeAvailableForOrganization(db, entry.Key, r.Organization.ID)
				require.NoError(t, err)
				assert.Equal(t, tc.wantAdded, available)

				available, err = models.IsIntakeAvailableForOrganization(db, entry.Key, other.ID)
				require.NoError(t, err)
				assert.Equal(t, tc.wantOther, available)
			})
		}
	})

	t.Run("an unknown key is not available", func(t *testing.T) {
		available, err := models.IsIntakeAvailableForOrganization(db, "does-not-exist", r.Organization.ID)
		require.NoError(t, err)
		assert.False(t, available)
	})

	t.Run("an unimplemented intake stays planned and accepts no companies", func(t *testing.T) {
		entry := createPlannedEntry(t, db, "acme-tracker")

		status := models.IntakeStatusBeta
		err := entry.Update(db, models.IntakeCatalogPatch{Status: &status}, nil)
		assert.ErrorIs(t, err, models.ErrIntakeCatalogNotImplemented)
		assert.ErrorIs(t, entry.AddOrganization(db, r.Organization.ID), models.ErrIntakeCatalogAccessNotSupported)

		stored, err := models.FindIntakeCatalogEntry(db, entry.Key)
		require.NoError(t, err)
		assert.Equal(t, models.IntakeStatusPlanned, stored.Status)
	})

	t.Run("an internal intake cannot be open to all companies", func(t *testing.T) {
		entry := withCatalogEntryState(t, db, models.FactoryIntakeSourceDatadog, models.IntakeStatusAlpha, false)

		enabled := true
		err := entry.Update(db, models.IntakeCatalogPatch{EnabledForAll: &enabled}, nil)
		assert.ErrorIs(t, err, models.ErrIntakeCatalogInternalForAll)
	})

	t.Run("moving an open beta to internal closes it", func(t *testing.T) {
		entry := withCatalogEntryState(t, db, models.FactoryIntakeSourceDatadog, models.IntakeStatusBeta, true)

		status := models.IntakeStatusAlpha
		require.NoError(t, entry.Update(db, models.IntakeCatalogPatch{Status: &status}, &r.Account.ID))

		stored, err := models.FindIntakeCatalogEntry(db, entry.Key)
		require.NoError(t, err)
		assert.Equal(t, models.IntakeStatusAlpha, stored.Status)
		assert.False(t, stored.EnabledForAll)
		assert.Equal(t, r.Account.Name, stored.UpdatedByName)
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
func withCatalogEntryState(t *testing.T, db *gorm.DB, key, status string, enabledForAll bool) *models.IntakeCatalogEntry {
	t.Helper()

	entry, err := models.FindIntakeCatalogEntry(db, key)
	require.NoError(t, err)
	original := *entry
	t.Cleanup(func() {
		require.NoError(t, db.Model(&models.IntakeCatalogEntry{}).Where("key = ?", key).Updates(map[string]any{
			"status":          original.Status,
			"enabled_for_all": original.EnabledForAll,
		}).Error)
	})

	require.NoError(t, db.Model(&models.IntakeCatalogEntry{}).Where("key = ?", key).Updates(map[string]any{
		"status":          status,
		"enabled_for_all": enabledForAll,
	}).Error)

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
