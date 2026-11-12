package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

func TestOrganizationEnterpriseFeatures(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	tx := database.Conn()

	t.Run("grant lookup and list", func(t *testing.T) {
		org := createEnterpriseFeatureOrg(t, "grant-lookup")
		actor := uuid.New()

		inserted, err := GrantEnterpriseFeatures(tx, org.ID, []string{"groups", "groups", "custom_roles"}, EnterpriseFeatureSourceAdmin, &actor)
		require.NoError(t, err)
		assert.Equal(t, int64(2), inserted)

		again, err := GrantEnterpriseFeatures(tx, org.ID, []string{"groups"}, EnterpriseFeatureSourceAutoSingleOrg, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(0), again)

		has, err := HasOrganizationEnterpriseFeature(tx, org.ID, "groups")
		require.NoError(t, err)
		assert.True(t, has)

		missing, err := HasOrganizationEnterpriseFeature(tx, org.ID, "audit_logs")
		require.NoError(t, err)
		assert.False(t, missing)

		none, err := HasOrganizationEnterpriseFeature(tx, uuid.Nil, "groups")
		require.NoError(t, err)
		assert.False(t, none)

		listed, err := ListOrganizationEnterpriseFeatures(tx, org.ID)
		require.NoError(t, err)
		require.Len(t, listed, 2)
		assert.Equal(t, "custom_roles", listed[0].FeatureKey)
		assert.Equal(t, "groups", listed[1].FeatureKey)
		assert.Equal(t, EnterpriseFeatureSourceAdmin, listed[0].Source)
		require.NotNil(t, listed[0].GrantedBy)
		assert.Equal(t, actor, *listed[0].GrantedBy)
	})

	t.Run("set replaces the grant set and keeps unchanged rows", func(t *testing.T) {
		org := createEnterpriseFeatureOrg(t, "grant-set")
		originalActor := uuid.New()
		_, err := GrantEnterpriseFeatures(tx, org.ID, []string{"groups", "custom_roles"}, EnterpriseFeatureSourceAutoSingleOrg, &originalActor)
		require.NoError(t, err)

		past := time.Now().Add(-2 * time.Hour)
		require.NoError(t, tx.Model(&OrganizationEnterpriseFeature{}).
			Where("organization_id = ? AND feature_key = ?", org.ID, "groups").
			Update("created_at", past).Error)

		nextActor := uuid.New()
		added, removed, err := SetOrganizationEnterpriseFeatures(tx, org.ID, []string{"groups", "audit_logs", "groups"}, nextActor)
		require.NoError(t, err)
		require.Len(t, added, 1)
		assert.Equal(t, "audit_logs", added[0].FeatureKey)
		assert.Equal(t, EnterpriseFeatureSourceAdmin, added[0].Source)
		require.NotNil(t, added[0].GrantedBy)
		assert.Equal(t, nextActor, *added[0].GrantedBy)
		require.Len(t, removed, 1)
		assert.Equal(t, "custom_roles", removed[0].FeatureKey)

		listed, err := ListOrganizationEnterpriseFeatures(tx, org.ID)
		require.NoError(t, err)
		require.Len(t, listed, 2)
		assert.Equal(t, "audit_logs", listed[0].FeatureKey)
		assert.Equal(t, "groups", listed[1].FeatureKey)
		assert.Equal(t, EnterpriseFeatureSourceAutoSingleOrg, listed[1].Source)
		require.NotNil(t, listed[1].GrantedBy)
		assert.Equal(t, originalActor, *listed[1].GrantedBy)
		assert.WithinDuration(t, past, listed[1].CreatedAt, time.Second)
	})

	t.Run("overlapping replacements do not merge grant sets", func(t *testing.T) {
		org := createEnterpriseFeatureOrg(t, "grant-overlap")
		db := database.Conn()

		holder := db.Begin()
		require.NoError(t, holder.Error)
		t.Cleanup(func() { _ = holder.Rollback() })
		require.NoError(t, holder.Exec("SELECT id FROM organizations WHERE id = ? FOR UPDATE", org.ID).Error)
		_, err := GrantEnterpriseFeatures(holder, org.ID, []string{"groups"}, EnterpriseFeatureSourceAdmin, nil)
		require.NoError(t, err)

		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("SET LOCAL lock_timeout = '200ms'").Error; err != nil {
				return err
			}
			_, _, err := SetOrganizationEnterpriseFeatures(tx, org.ID, []string{"audit_logs"}, uuid.New())
			return err
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lock timeout")

		require.NoError(t, holder.Commit().Error)

		_, removed, err := SetOrganizationEnterpriseFeatures(db, org.ID, []string{"audit_logs"}, uuid.New())
		require.NoError(t, err)
		require.Len(t, removed, 1)
		assert.Equal(t, "groups", removed[0].FeatureKey)

		listed, err := ListOrganizationEnterpriseFeatures(db, org.ID)
		require.NoError(t, err)
		require.Len(t, listed, 1)
		assert.Equal(t, "audit_logs", listed[0].FeatureKey)
	})

	t.Run("lists and counts exclude deleted organizations", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		first := createEnterpriseFeatureOrg(t, "grant-list-a")
		second := createEnterpriseFeatureOrg(t, "grant-list-b")
		deleted := createEnterpriseFeatureOrg(t, "grant-list-deleted")
		_, err := GrantEnterpriseFeatures(tx, first.ID, []string{"groups"}, EnterpriseFeatureSourceAdmin, nil)
		require.NoError(t, err)
		_, err = GrantEnterpriseFeatures(tx, second.ID, []string{"groups", "custom_roles"}, EnterpriseFeatureSourceAdmin, nil)
		require.NoError(t, err)
		_, err = GrantEnterpriseFeatures(tx, deleted.ID, []string{"groups"}, EnterpriseFeatureSourceAdmin, nil)
		require.NoError(t, err)
		require.NoError(t, SoftDeleteOrganization(deleted.ID.String()))

		ids, total, err := ListOrganizationsWithEnterpriseFeature(tx, "groups", 1, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		require.Len(t, ids, 1)

		rest, total, err := ListOrganizationsWithEnterpriseFeature(tx, "groups", 10, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		require.Len(t, rest, 1)
		assert.NotEqual(t, ids[0], rest[0])

		counts, err := CountOrganizationsByEnterpriseFeature(tx)
		require.NoError(t, err)
		assert.Equal(t, int64(2), counts["groups"])
		assert.Equal(t, int64(1), counts["custom_roles"])
	})

	t.Run("prune and delete return the removed rows", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		org := createEnterpriseFeatureOrg(t, "grant-prune")
		_, err := GrantEnterpriseFeatures(tx, org.ID, []string{"groups", "custom_roles", "audit_logs"}, EnterpriseFeatureSourceAdmin, nil)
		require.NoError(t, err)

		removed, err := PruneEnterpriseFeaturesNotIn(tx, []string{"custom_roles"})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"groups", "audit_logs"}, featureKeys(removed))

		remaining, err := ListOrganizationEnterpriseFeatures(tx, org.ID)
		require.NoError(t, err)
		require.Len(t, remaining, 1)
		assert.Equal(t, "custom_roles", remaining[0].FeatureKey)

		all, err := DeleteAllEnterpriseFeatures(tx)
		require.NoError(t, err)
		require.NotEmpty(t, all)

		left, err := ListOrganizationEnterpriseFeatures(tx, org.ID)
		require.NoError(t, err)
		assert.Empty(t, left)
	})

	t.Run("empty prune deletes every grant", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		org := createEnterpriseFeatureOrg(t, "grant-prune-empty")
		_, err := GrantEnterpriseFeatures(tx, org.ID, []string{"groups"}, EnterpriseFeatureSourceAdmin, nil)
		require.NoError(t, err)

		removed, err := PruneEnterpriseFeaturesNotIn(tx, nil)
		require.NoError(t, err)
		require.NotEmpty(t, removed)

		has, err := HasOrganizationEnterpriseFeature(tx, org.ID, "groups")
		require.NoError(t, err)
		assert.False(t, has)
	})

	t.Run("rejects an unknown source and a bad feature key", func(t *testing.T) {
		org := createEnterpriseFeatureOrg(t, "grant-check")
		err := tx.Create(&OrganizationEnterpriseFeature{
			OrganizationID: org.ID,
			FeatureKey:     "groups",
			Source:         "billing",
			CreatedAt:      time.Now(),
		}).Error
		require.Error(t, err)

		err = tx.Create(&OrganizationEnterpriseFeature{
			OrganizationID: org.ID,
			FeatureKey:     "Groups",
			Source:         EnterpriseFeatureSourceAdmin,
			CreatedAt:      time.Now(),
		}).Error
		require.Error(t, err)
	})
}

func TestInstallationLicenseActivations(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	tx := database.Conn()

	t.Run("one open activation and close once", func(t *testing.T) {
		require.Nil(t, openActivation(t, tx))

		licenseID := uuid.New()
		activation := &InstallationLicenseActivation{
			LicenseID:         licenseID,
			LicenseFeatures:   []string{"custom_roles", "groups"},
			OrganizationCount: 0,
		}
		inserted, err := InsertOpenLicenseActivation(tx, activation)
		require.NoError(t, err)
		assert.True(t, inserted)
		assert.NotZero(t, activation.ID)
		assert.False(t, activation.ActivatedAt.IsZero())

		again := &InstallationLicenseActivation{LicenseID: uuid.New(), LicenseFeatures: []string{"groups"}, OrganizationCount: 1}
		inserted, err = InsertOpenLicenseActivation(tx, again)
		require.NoError(t, err)
		assert.False(t, inserted)

		locked, err := LockOpenLicenseActivationForShare(tx)
		require.NoError(t, err)
		require.NotNil(t, locked)
		assert.Equal(t, activation.ID, locked.ID)
		assert.Equal(t, []string{"custom_roles", "groups"}, []string(locked.LicenseFeatures))

		require.NoError(t, SetLicenseActivationResult(tx, activation.ID, 3, true))
		open, err := FindOpenLicenseActivation(tx)
		require.NoError(t, err)
		require.NotNil(t, open)
		assert.Equal(t, 3, open.OrganizationCount)
		assert.True(t, open.AutoGranted)

		closed, err := CloseLicenseActivation(tx, activation.ID, LicenseDeactivationReasonReplaced)
		require.NoError(t, err)
		assert.True(t, closed)
		closed, err = CloseLicenseActivation(tx, activation.ID, LicenseDeactivationReasonExpired)
		require.NoError(t, err)
		assert.False(t, closed)

		open, err = FindOpenLicenseActivation(tx)
		require.NoError(t, err)
		assert.Nil(t, open)

		rows, total, err := ListLicenseActivations(tx, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, rows, 1)
		require.NotNil(t, rows[0].DeactivationReason)
		assert.Equal(t, LicenseDeactivationReasonReplaced, *rows[0].DeactivationReason)
		assert.NotNil(t, rows[0].DeactivatedAt)
	})

	t.Run("lists newest activations first and stores removed grants", func(t *testing.T) {
		olderID := insertClosedActivation(t, tx, time.Now().Add(-2*time.Hour))
		newer := &InstallationLicenseActivation{LicenseID: uuid.New(), LicenseFeatures: []string{"groups"}, OrganizationCount: 1}
		inserted, err := InsertOpenLicenseActivation(tx, newer)
		require.NoError(t, err)
		require.True(t, inserted)
		closed, err := CloseLicenseActivation(tx, newer.ID, LicenseDeactivationReasonRemoved)
		require.NoError(t, err)
		require.True(t, closed)

		actor := uuid.New()
		orgID := uuid.New()
		createdAt := time.Now().Add(-time.Minute).UTC().Truncate(time.Microsecond)
		removedAt := time.Now().UTC().Truncate(time.Microsecond)
		require.NoError(t, AppendRemovedGrants(tx, newer.ID, []RemovedEnterpriseGrant{{
			OrganizationID: orgID,
			FeatureKey:     "groups",
			Source:         EnterpriseFeatureSourceAdmin,
			GrantedBy:      &actor,
			CreatedAt:      createdAt,
			RemovedAt:      removedAt,
			RemovedBy:      EnterpriseFeatureRemovedByDeactivation,
		}}))
		require.NoError(t, AppendRemovedGrants(tx, newer.ID, nil))

		rows, total, err := ListLicenseActivations(tx, 1, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(2))
		require.Len(t, rows, 1)
		assert.Equal(t, newer.ID, rows[0].ID)
		require.Len(t, rows[0].RemovedGrants, 1)
		grant := rows[0].RemovedGrants[0]
		assert.Equal(t, orgID, grant.OrganizationID)
		assert.Equal(t, "groups", grant.FeatureKey)
		assert.Equal(t, EnterpriseFeatureSourceAdmin, grant.Source)
		require.NotNil(t, grant.GrantedBy)
		assert.Equal(t, actor, *grant.GrantedBy)
		assert.Equal(t, EnterpriseFeatureRemovedByDeactivation, grant.RemovedBy)
		assert.WithinDuration(t, removedAt, grant.RemovedAt, time.Second)

		withoutActor := RemovedEnterpriseGrant{
			OrganizationID: orgID,
			FeatureKey:     "custom_roles",
			Source:         EnterpriseFeatureSourceAutoSingleOrg,
			RemovedAt:      removedAt,
			RemovedBy:      EnterpriseFeatureRemovedByActivation,
			CreatedAt:      createdAt,
		}
		require.NoError(t, AppendRemovedGrants(tx, olderID, []RemovedEnterpriseGrant{withoutActor}))
		rows, _, err = ListLicenseActivations(tx, 10, 0)
		require.NoError(t, err)
		var stored *RemovedEnterpriseGrant
		for _, row := range rows {
			if row.ID != olderID {
				continue
			}
			require.Len(t, row.RemovedGrants, 1)
			stored = &row.RemovedGrants[0]
		}
		require.NotNil(t, stored)
		assert.Nil(t, stored.GrantedBy)
	})

	t.Run("counts active organizations including suspended ones", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		count, err := CountOrganizations(tx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
		only, err := FindOnlyOrganization(tx)
		require.NoError(t, err)
		assert.Nil(t, only)

		org := createEnterpriseFeatureOrg(t, "activation-only")
		suspendedAt := time.Now()
		require.NoError(t, tx.Model(org).Update("suspended_at", suspendedAt).Error)

		count, err = CountOrganizations(tx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
		only, err = FindOnlyOrganization(tx)
		require.NoError(t, err)
		require.NotNil(t, only)
		assert.Equal(t, org.ID, only.ID)

		_ = createEnterpriseFeatureOrg(t, "activation-second")
		count, err = CountOrganizations(tx)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
		only, err = FindOnlyOrganization(tx)
		require.NoError(t, err)
		assert.Nil(t, only)

		deleted := createEnterpriseFeatureOrg(t, "activation-deleted")
		require.NoError(t, SoftDeleteOrganization(deleted.ID.String()))
		count, err = CountOrganizations(tx)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})
}

func createEnterpriseFeatureOrg(t *testing.T, name string) *Organization {
	t.Helper()
	org, err := CreateOrganization(name, "")
	require.NoError(t, err)
	return org
}

func featureKeys(rows []OrganizationEnterpriseFeature) []string {
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, row.FeatureKey)
	}
	return keys
}

func openActivation(t *testing.T, tx *gorm.DB) *InstallationLicenseActivation {
	t.Helper()
	row, err := FindOpenLicenseActivation(tx)
	require.NoError(t, err)
	return row
}

func insertClosedActivation(t *testing.T, tx *gorm.DB, activatedAt time.Time) int64 {
	t.Helper()
	activation := &InstallationLicenseActivation{
		LicenseID:         uuid.New(),
		LicenseFeatures:   []string{"groups"},
		OrganizationCount: 1,
	}
	inserted, err := InsertOpenLicenseActivation(tx, activation)
	require.NoError(t, err)
	require.True(t, inserted)
	require.NoError(t, tx.Model(&InstallationLicenseActivation{}).Where("id = ?", activation.ID).Update("activated_at", activatedAt).Error)
	closed, err := CloseLicenseActivation(tx, activation.ID, LicenseDeactivationReasonExpired)
	require.NoError(t, err)
	require.True(t, closed)
	return activation.ID
}
