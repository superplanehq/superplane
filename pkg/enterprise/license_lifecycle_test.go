package enterprise

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestLicenseLifecycle(t *testing.T) {
	lifecycle := NewLicenseLifecycle(database.Conn)

	t.Run("no open row and an inactive license does nothing", func(t *testing.T) {
		for _, state := range []licensing.State{
			licensing.StateNone,
			licensing.StateExpired,
			licensing.StateRevoked,
			licensing.StateInvalid,
			licensing.StateNotYetValid,
		} {
			resetEnterpriseGrants(t)
			require.NoError(t, lifecycle.HandleStatus(context.Background(), licensing.Status{State: state}))
			assert.Nil(t, openActivation(t))
		}
	})

	t.Run("one organization receives every licensed feature", func(t *testing.T) {
		resetEnterpriseGrants(t)
		org := createLifecycleOrg(t, "lifecycle-one")
		licenseID := uuid.New()
		require.NoError(t, lifecycle.HandleStatus(context.Background(), activeStatus(licenseID, licensing.FeatureGroups, licensing.FeatureCustomRoles)))

		open := openActivation(t)
		require.NotNil(t, open)
		assert.Equal(t, licenseID, open.LicenseID)
		assert.Equal(t, []string{"custom_roles", "groups"}, []string(open.LicenseFeatures))
		assert.Equal(t, 1, open.OrganizationCount)
		assert.True(t, open.AutoGranted)
		grants := grantsByKey(t, org.ID)
		require.Len(t, grants, 2)
		assert.Equal(t, models.EnterpriseFeatureSourceAutoSingleOrg, grants["custom_roles"].Source)
		assert.Equal(t, models.EnterpriseFeatureSourceAutoSingleOrg, grants["groups"].Source)
		assert.Nil(t, grants["groups"].GrantedBy)
	})

	t.Run("activation keeps an existing grant and prunes the rest", func(t *testing.T) {
		resetEnterpriseGrants(t)
		org := createLifecycleOrg(t, "lifecycle-keep")
		actor := uuid.New()
		_, err := models.GrantEnterpriseFeatures(database.Conn(), org.ID, []string{"custom_roles", "audit_logs"}, models.EnterpriseFeatureSourceAdmin, &actor)
		require.NoError(t, err)
		past := time.Now().Add(-2 * time.Hour)
		require.NoError(t, database.Conn().Model(&models.OrganizationEnterpriseFeature{}).
			Where("organization_id = ? AND feature_key = ?", org.ID, "custom_roles").
			Update("created_at", past).Error)

		require.NoError(t, lifecycle.HandleStatus(context.Background(), activeStatus(uuid.New(), licensing.FeatureCustomRoles, licensing.FeatureGroups)))

		grants := grantsByKey(t, org.ID)
		require.Len(t, grants, 2)
		assert.Equal(t, models.EnterpriseFeatureSourceAdmin, grants["custom_roles"].Source)
		require.NotNil(t, grants["custom_roles"].GrantedBy)
		assert.Equal(t, actor, *grants["custom_roles"].GrantedBy)
		assert.WithinDuration(t, past, grants["custom_roles"].CreatedAt, time.Second)
		assert.Equal(t, models.EnterpriseFeatureSourceAutoSingleOrg, grants["groups"].Source)
		open := openActivation(t)
		require.NotNil(t, open)
		assert.True(t, open.AutoGranted)
		require.Len(t, open.RemovedGrants, 1)
		assert.Equal(t, "audit_logs", open.RemovedGrants[0].FeatureKey)
		assert.Equal(t, models.EnterpriseFeatureRemovedByActivation, open.RemovedGrants[0].RemovedBy)
	})

	t.Run("many organizations receive no grants", func(t *testing.T) {
		resetEnterpriseGrants(t)
		created := createLifecycleOrgs(t, 3)
		require.NoError(t, lifecycle.HandleStatus(context.Background(), activeStatus(uuid.New(), licensing.FeatureGroups)))

		open := openActivation(t)
		require.NotNil(t, open)
		assert.Equal(t, 3, open.OrganizationCount)
		assert.False(t, open.AutoGranted)
		for _, org := range created {
			assert.Empty(t, grantsByKey(t, org.ID))
		}
	})

	t.Run("zero organizations stay without grants after the first organization is created", func(t *testing.T) {
		resetEnterpriseGrants(t)
		licenseID := uuid.New()
		status := activeStatus(licenseID, licensing.FeatureGroups)
		require.NoError(t, lifecycle.HandleStatus(context.Background(), status))
		open := openActivation(t)
		require.NotNil(t, open)
		assert.Equal(t, 0, open.OrganizationCount)

		org := createLifecycleOrg(t, "lifecycle-later")
		require.NoError(t, lifecycle.HandleStatus(context.Background(), status))
		assert.Empty(t, grantsByKey(t, org.ID))
		assert.Equal(t, open.ID, openActivation(t).ID)
		assert.Equal(t, 0, openActivation(t).OrganizationCount)
	})

	t.Run("the same license does not restore a grant an administrator removed", func(t *testing.T) {
		resetEnterpriseGrants(t)
		org := createLifecycleOrg(t, "lifecycle-removed")
		licenseID := uuid.New()
		status := activeStatus(licenseID, licensing.FeatureCustomRoles, licensing.FeatureGroups)
		require.NoError(t, lifecycle.HandleStatus(context.Background(), status))
		require.NoError(t, database.Conn().
			Where("organization_id = ? AND feature_key = ?", org.ID, "groups").
			Delete(&models.OrganizationEnterpriseFeature{}).Error)
		openID := openActivation(t).ID

		require.NoError(t, lifecycle.HandleStatus(context.Background(), status))
		assert.Equal(t, openID, openActivation(t).ID)
		grants := grantsByKey(t, org.ID)
		_, hasGroups := grants["groups"]
		assert.False(t, hasGroups)
		assert.Contains(t, grants, "custom_roles")
	})

	t.Run("concurrent activation inserts one open row", func(t *testing.T) {
		resetEnterpriseGrants(t)
		org := createLifecycleOrg(t, "lifecycle-concurrent")
		status := activeStatus(uuid.New(), licensing.FeatureCustomRoles, licensing.FeatureGroups)
		runConcurrent(t, func() error {
			return lifecycle.HandleStatus(context.Background(), status)
		})

		rows := activationRows(t)
		require.Len(t, rows, 1)
		assert.Nil(t, rows[0].DeactivatedAt)
		assert.Len(t, grantsByKey(t, org.ID), 2)
	})

	t.Run("replacement removes only features the new license does not contain", func(t *testing.T) {
		resetEnterpriseGrants(t)
		created := createLifecycleOrgs(t, 3)
		first := activeStatus(uuid.New(), licensing.FeatureCustomRoles, licensing.FeatureGroups)
		require.NoError(t, lifecycle.HandleStatus(context.Background(), first))
		actor := uuid.New()
		past := time.Now().Add(-2 * time.Hour)
		for _, org := range created {
			_, err := models.GrantEnterpriseFeatures(database.Conn(), org.ID, []string{"custom_roles", "groups"}, models.EnterpriseFeatureSourceAdmin, &actor)
			require.NoError(t, err)
		}
		require.NoError(t, database.Conn().Model(&models.OrganizationEnterpriseFeature{}).
			Where("feature_key = ?", "custom_roles").
			Update("created_at", past).Error)

		second := activeStatus(uuid.New(), licensing.FeatureCustomRoles)
		require.NoError(t, lifecycle.HandleStatus(context.Background(), second))

		rows := activationRows(t)
		require.Len(t, rows, 2)
		var closed, open *models.InstallationLicenseActivation
		for i := range rows {
			if rows[i].DeactivatedAt == nil {
				open = &rows[i]
				continue
			}
			closed = &rows[i]
		}
		require.NotNil(t, closed)
		require.NotNil(t, open)
		require.NotNil(t, closed.DeactivationReason)
		assert.Equal(t, models.LicenseDeactivationReasonReplaced, *closed.DeactivationReason)
		assert.Equal(t, second.License.ID, open.LicenseID)
		assert.False(t, open.AutoGranted)
		assert.Empty(t, closed.RemovedGrants)
		require.Len(t, open.RemovedGrants, len(created))
		for _, grant := range open.RemovedGrants {
			assert.Equal(t, "groups", grant.FeatureKey)
			assert.Equal(t, models.EnterpriseFeatureRemovedByActivation, grant.RemovedBy)
		}
		for _, org := range created {
			grants := grantsByKey(t, org.ID)
			require.Len(t, grants, 1)
			assert.Equal(t, models.EnterpriseFeatureSourceAdmin, grants["custom_roles"].Source)
			require.NotNil(t, grants["custom_roles"].GrantedBy)
			assert.Equal(t, actor, *grants["custom_roles"].GrantedBy)
			assert.WithinDuration(t, past, grants["custom_roles"].CreatedAt, time.Second)
		}
	})

	t.Run("replacement gives new features only to a single organization", func(t *testing.T) {
		resetEnterpriseGrants(t)
		org := createLifecycleOrg(t, "lifecycle-new-feature")
		require.NoError(t, lifecycle.HandleStatus(context.Background(), activeStatus(uuid.New(), licensing.FeatureCustomRoles, licensing.FeatureGroups)))
		past := time.Now().Add(-2 * time.Hour)
		require.NoError(t, database.Conn().Model(&models.OrganizationEnterpriseFeature{}).
			Where("organization_id = ?", org.ID).
			Update("created_at", past).Error)

		require.NoError(t, lifecycle.HandleStatus(context.Background(), activeStatus(uuid.New(), licensing.FeatureCustomRoles, licensing.FeatureGroups, licensing.FeatureAuditLogs)))

		grants := grantsByKey(t, org.ID)
		require.Len(t, grants, 3)
		assert.Equal(t, models.EnterpriseFeatureSourceAutoSingleOrg, grants["audit_logs"].Source)
		assert.Nil(t, grants["audit_logs"].GrantedBy)
		assert.WithinDuration(t, past, grants["custom_roles"].CreatedAt, time.Second)
		assert.WithinDuration(t, past, grants["groups"].CreatedAt, time.Second)
		assert.True(t, openActivation(t).AutoGranted)
	})

	t.Run("replacement does not give a new feature to many organizations", func(t *testing.T) {
		resetEnterpriseGrants(t)
		created := createLifecycleOrgs(t, 3)
		require.NoError(t, lifecycle.HandleStatus(context.Background(), activeStatus(uuid.New(), licensing.FeatureCustomRoles)))
		for _, org := range created {
			_, err := models.GrantEnterpriseFeatures(database.Conn(), org.ID, []string{"custom_roles"}, models.EnterpriseFeatureSourceAdmin, nil)
			require.NoError(t, err)
		}

		require.NoError(t, lifecycle.HandleStatus(context.Background(), activeStatus(uuid.New(), licensing.FeatureCustomRoles, licensing.FeatureAuditLogs)))
		for _, org := range created {
			_, hasAudit := grantsByKey(t, org.ID)["audit_logs"]
			assert.False(t, hasAudit)
		}
	})

	t.Run("installing the same license after removal grants the single organization again", func(t *testing.T) {
		resetEnterpriseGrants(t)
		org := createLifecycleOrg(t, "lifecycle-reinstall")
		licenseID := uuid.New()
		status := activeStatus(licenseID, licensing.FeatureGroups)
		require.NoError(t, lifecycle.HandleStatus(context.Background(), status))
		require.NoError(t, lifecycle.HandleStatus(context.Background(), licensing.Status{State: licensing.StateNone}))
		assert.Empty(t, grantsByKey(t, org.ID))

		require.NoError(t, lifecycle.HandleStatus(context.Background(), status))
		rows := activationRows(t)
		require.Len(t, rows, 2)
		open := openActivation(t)
		require.NotNil(t, open)
		assert.Equal(t, licenseID, open.LicenseID)
		assert.True(t, open.AutoGranted)
		assert.Contains(t, grantsByKey(t, org.ID), "groups")
		var removed bool
		for _, row := range rows {
			if row.DeactivationReason != nil && *row.DeactivationReason == models.LicenseDeactivationReasonRemoved {
				removed = true
			}
		}
		assert.True(t, removed)
	})

	t.Run("expiry revocation and removal delete every grant", func(t *testing.T) {
		cases := []struct {
			state  licensing.State
			reason string
		}{
			{state: licensing.StateExpired, reason: models.LicenseDeactivationReasonExpired},
			{state: licensing.StateRevoked, reason: models.LicenseDeactivationReasonRevoked},
			{state: licensing.StateNone, reason: models.LicenseDeactivationReasonRemoved},
		}
		for _, tc := range cases {
			t.Run(string(tc.state), func(t *testing.T) {
				resetEnterpriseGrants(t)
				org := createLifecycleOrg(t, "lifecycle-end")
				require.NoError(t, lifecycle.HandleStatus(context.Background(), activeStatus(uuid.New(), licensing.FeatureGroups, licensing.FeatureCustomRoles)))
				removedAt := time.Now().UTC().Truncate(time.Millisecond)
				lifecycle.now = func() time.Time { return removedAt }
				t.Cleanup(func() { lifecycle.now = time.Now })

				loadedID := uuid.New()
				require.NoError(t, lifecycle.HandleStatus(context.Background(), licensing.Status{
					State:   tc.state,
					License: &licensing.License{ID: loadedID, Features: []licensing.Feature{licensing.FeatureGroups}},
				}))

				assert.Nil(t, openActivation(t))
				assert.Empty(t, grantsByKey(t, org.ID))
				rows := activationRows(t)
				require.Len(t, rows, 1)
				require.NotNil(t, rows[0].DeactivationReason)
				assert.Equal(t, tc.reason, *rows[0].DeactivationReason)
				require.Len(t, rows[0].RemovedGrants, 2)
				for _, grant := range rows[0].RemovedGrants {
					assert.Equal(t, models.EnterpriseFeatureRemovedByDeactivation, grant.RemovedBy)
					assert.WithinDuration(t, removedAt, grant.RemovedAt, time.Second)
				}
			})
		}
	})

	t.Run("invalid and not yet valid licenses keep grants", func(t *testing.T) {
		for _, state := range []licensing.State{licensing.StateInvalid, licensing.StateNotYetValid} {
			resetEnterpriseGrants(t)
			org := createLifecycleOrg(t, "lifecycle-keep-state")
			status := activeStatus(uuid.New(), licensing.FeatureGroups)
			require.NoError(t, lifecycle.HandleStatus(context.Background(), status))
			openID := openActivation(t).ID

			require.NoError(t, lifecycle.HandleStatus(context.Background(), licensing.Status{State: state, License: status.License}))
			assert.Equal(t, openID, openActivation(t).ID)
			assert.Nil(t, openActivation(t).DeactivatedAt)
			assert.Contains(t, grantsByKey(t, org.ID), "groups")
		}
	})

	t.Run("concurrent deactivation closes the row once", func(t *testing.T) {
		resetEnterpriseGrants(t)
		org := createLifecycleOrg(t, "lifecycle-close-once")
		require.NoError(t, lifecycle.HandleStatus(context.Background(), activeStatus(uuid.New(), licensing.FeatureGroups, licensing.FeatureCustomRoles)))
		runConcurrent(t, func() error {
			return lifecycle.HandleStatus(context.Background(), licensing.Status{State: licensing.StateExpired})
		})

		rows := activationRows(t)
		require.Len(t, rows, 1)
		require.NotNil(t, rows[0].DeactivationReason)
		assert.Equal(t, models.LicenseDeactivationReasonExpired, *rows[0].DeactivationReason)
		assert.Empty(t, grantsByKey(t, org.ID))
		assert.Len(t, rows[0].RemovedGrants, 2)
	})

	t.Run("a development license does not write an activation", func(t *testing.T) {
		resetEnterpriseGrants(t)
		_ = createLifecycleOrg(t, "lifecycle-dev")
		require.NoError(t, lifecycle.HandleStatus(context.Background(), licensing.Status{
			State: licensing.StateActive,
			License: &licensing.License{
				ID:       uuid.New(),
				KeyID:    licensing.DevelopmentKeyID,
				Features: []licensing.Feature{licensing.FeatureGroups},
			},
		}))
		assert.Empty(t, activationRows(t))
	})
}

func resetEnterpriseGrants(t *testing.T) {
	t.Helper()
	require.NoError(t, database.TruncateTables())
}

func createLifecycleOrg(t *testing.T, name string) *models.Organization {
	t.Helper()
	org, err := models.CreateOrganization(name, "")
	require.NoError(t, err)
	return org
}

func createLifecycleOrgs(t *testing.T, count int) []*models.Organization {
	t.Helper()
	orgs := make([]*models.Organization, 0, count)
	for i := 0; i < count; i++ {
		org, err := models.CreateOrganization(uuid.NewString(), "")
		require.NoError(t, err)
		orgs = append(orgs, org)
	}
	return orgs
}

func activeStatus(id uuid.UUID, features ...licensing.Feature) licensing.Status {
	return licensing.Status{
		State: licensing.StateActive,
		License: &licensing.License{
			ID:       id,
			KeyID:    "test-key",
			Features: features,
		},
	}
}

func openActivation(t *testing.T) *models.InstallationLicenseActivation {
	t.Helper()
	row, err := models.FindOpenLicenseActivation(database.Conn())
	require.NoError(t, err)
	return row
}

func activationRows(t *testing.T) []models.InstallationLicenseActivation {
	t.Helper()
	rows, _, err := models.ListLicenseActivations(database.Conn(), 0, 0)
	require.NoError(t, err)
	return rows
}

func grantsByKey(t *testing.T, orgID uuid.UUID) map[string]models.OrganizationEnterpriseFeature {
	t.Helper()
	rows, err := models.ListOrganizationEnterpriseFeatures(database.Conn(), orgID)
	require.NoError(t, err)
	byKey := make(map[string]models.OrganizationEnterpriseFeature, len(rows))
	for _, row := range rows {
		byKey[row.FeatureKey] = row
	}
	return byKey
}

func runConcurrent(t *testing.T, fn func() error) {
	t.Helper()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- fn()
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}
