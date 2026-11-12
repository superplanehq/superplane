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
	"gorm.io/gorm"
)

type staticLicense struct {
	status licensing.Status
}

func (s staticLicense) Status() licensing.Status { return s.status }

func TestOrganizationEnterpriseFeatureSet(t *testing.T) {
	ctx := context.Background()
	actor := createFeatureAccount(t)

	t.Run("enables a licensed feature and records the administrator", func(t *testing.T) {
		resetEnterpriseGrants(t)
		org := createLifecycleOrg(t, "feature-enable")
		openTestLicense(t, "custom_roles", "groups")
		features := NewOrganizationFeatures(staticLicense{status: activeStatus(uuid.New(), licensing.FeatureCustomRoles, licensing.FeatureGroups)})

		added, removed, err := features.Set(ctx, org.ID, []string{"groups", "groups"}, actor.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"groups"}, added)
		assert.Empty(t, removed)

		grant := grantsByKey(t, org.ID)["groups"]
		assert.Equal(t, models.EnterpriseFeatureSourceAdmin, grant.Source)
		require.NotNil(t, grant.GrantedBy)
		assert.Equal(t, actor.ID, *grant.GrantedBy)
	})

	t.Run("refuses a feature the license does not contain", func(t *testing.T) {
		resetEnterpriseGrants(t)
		org := createLifecycleOrg(t, "feature-unlicensed")
		openTestLicense(t, "custom_roles")
		features := NewOrganizationFeatures(staticLicense{})

		_, _, err := features.Set(ctx, org.ID, []string{"groups"}, actor.ID)
		var unlicensed UnlicensedFeatureError
		require.ErrorAs(t, err, &unlicensed)
		assert.Equal(t, "groups", unlicensed.Key)
		assert.Empty(t, grantsByKey(t, org.ID))

		_, _, err = features.Set(ctx, org.ID, []string{"not_a_feature"}, actor.ID)
		require.ErrorAs(t, err, &unlicensed)
		assert.Equal(t, "not_a_feature", unlicensed.Key)
	})

	t.Run("refuses every enable without an open license and allows disable", func(t *testing.T) {
		resetEnterpriseGrants(t)
		org := createLifecycleOrg(t, "feature-no-license")
		_, err := models.GrantEnterpriseFeatures(database.Conn(), org.ID, []string{"groups"}, models.EnterpriseFeatureSourceAdmin, &actor.ID)
		require.NoError(t, err)
		features := NewOrganizationFeatures(staticLicense{})

		_, _, err = features.Set(ctx, org.ID, []string{"groups", "custom_roles"}, actor.ID)
		require.ErrorIs(t, err, ErrActiveEnterpriseLicenseRequired)
		assert.Contains(t, grantsByKey(t, org.ID), "groups")

		added, removed, err := features.Set(ctx, org.ID, nil, actor.ID)
		require.NoError(t, err)
		assert.Empty(t, added)
		assert.Equal(t, []string{"groups"}, removed)
		assert.Empty(t, grantsByKey(t, org.ID))
	})

	t.Run("keeps the source and time of a grant that stays", func(t *testing.T) {
		resetEnterpriseGrants(t)
		org := createLifecycleOrg(t, "feature-keep")
		openTestLicense(t, "custom_roles", "groups")
		_, err := models.GrantEnterpriseFeatures(database.Conn(), org.ID, []string{"groups"}, models.EnterpriseFeatureSourceAutoSingleOrg, nil)
		require.NoError(t, err)
		past := time.Now().Add(-2 * time.Hour)
		require.NoError(t, database.Conn().Model(&models.OrganizationEnterpriseFeature{}).
			Where("organization_id = ? AND feature_key = ?", org.ID, "groups").
			Update("created_at", past).Error)

		features := NewOrganizationFeatures(staticLicense{status: activeStatus(uuid.New(), licensing.FeatureCustomRoles, licensing.FeatureGroups)})
		added, removed, err := features.Set(ctx, org.ID, []string{"groups", "custom_roles"}, actor.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"custom_roles"}, added)
		assert.Empty(t, removed)

		kept := grantsByKey(t, org.ID)["groups"]
		assert.Equal(t, models.EnterpriseFeatureSourceAutoSingleOrg, kept.Source)
		assert.Nil(t, kept.GrantedBy)
		assert.WithinDuration(t, past, kept.CreatedAt, time.Second)

		created := grantsByKey(t, org.ID)["custom_roles"]
		assert.Equal(t, models.EnterpriseFeatureSourceAdmin, created.Source)
		require.NotNil(t, created.GrantedBy)
		assert.Equal(t, actor.ID, *created.GrantedBy)
	})

	t.Run("a concurrent deactivation leaves no grant outside the license", func(t *testing.T) {
		features := NewOrganizationFeatures(staticLicense{})
		lifecycle := NewLicenseLifecycle(database.Conn)
		for range 10 {
			resetEnterpriseGrants(t)
			org := createLifecycleOrg(t, "feature-race")
			openTestLicense(t, "custom_roles", "groups")

			var setErr, lifeErr error
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				_, _, setErr = features.Set(ctx, org.ID, []string{"custom_roles", "groups"}, actor.ID)
			}()
			go func() {
				defer wg.Done()
				lifeErr = lifecycle.HandleStatus(ctx, licensing.Status{State: licensing.StateExpired})
			}()
			wg.Wait()

			require.NoError(t, lifeErr)
			if setErr != nil {
				require.ErrorIs(t, setErr, ErrActiveEnterpriseLicenseRequired)
			}
			assert.Nil(t, openActivation(t))
			assert.Empty(t, grantsByKey(t, org.ID))
		}
	})

	t.Run("unknown organization is not found", func(t *testing.T) {
		resetEnterpriseGrants(t)
		features := NewOrganizationFeatures(staticLicense{})
		_, _, err := features.Set(ctx, uuid.New(), nil, actor.ID)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func TestOrganizationEnterpriseFeatureListAndEffective(t *testing.T) {
	ctx := context.Background()
	resetEnterpriseGrants(t)
	org := createLifecycleOrg(t, "feature-list")
	actor := createFeatureAccount(t)
	_, err := models.GrantEnterpriseFeatures(database.Conn(), org.ID, []string{"groups"}, models.EnterpriseFeatureSourceAdmin, &actor.ID)
	require.NoError(t, err)

	active := NewOrganizationFeatures(staticLicense{status: activeStatus(uuid.New(), licensing.FeatureCustomRoles, licensing.FeatureGroups)})
	views, licenseActive, err := active.List(ctx, org.ID)
	require.NoError(t, err)
	assert.True(t, licenseActive)
	require.Len(t, views, 2)
	assert.Equal(t, "custom_roles", views[0].Key)
	assert.True(t, views[0].Licensed)
	assert.False(t, views[0].Granted)
	assert.False(t, views[0].Effective)
	assert.Equal(t, "groups", views[1].Key)
	assert.True(t, views[1].Licensed)
	assert.True(t, views[1].Granted)
	assert.True(t, views[1].Effective)
	assert.Equal(t, actor.Name, views[1].GrantedByName)

	effective, err := active.Effective(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, []licensing.Feature{licensing.FeatureGroups}, effective)

	expired := NewOrganizationFeatures(staticLicense{status: licensing.Status{
		State:   licensing.StateExpired,
		License: &licensing.License{Features: []licensing.Feature{licensing.FeatureGroups, licensing.FeatureCustomRoles}},
	}})
	views, licenseActive, err = expired.List(ctx, org.ID)
	require.NoError(t, err)
	assert.False(t, licenseActive)
	require.Len(t, views, 1)
	assert.Equal(t, "groups", views[0].Key)
	assert.False(t, views[0].Licensed)
	assert.True(t, views[0].Granted)
	assert.False(t, views[0].Effective)

	development := NewOrganizationFeatures(staticLicense{status: licensing.Status{
		State: licensing.StateActive,
		License: &licensing.License{
			KeyID:    licensing.DevelopmentKeyID,
			Features: []licensing.Feature{licensing.FeatureCustomRoles},
		},
	}})
	effective, err = development.Effective(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, []licensing.Feature{licensing.FeatureCustomRoles}, effective)
	effective, err = active.Effective(ctx, uuid.Nil)
	require.NoError(t, err)
	assert.Empty(t, effective)
}

func createFeatureAccount(t *testing.T) *models.Account {
	t.Helper()
	account, err := models.CreateAccount("Ada Admin", "ada-"+uuid.NewString()+"@example.com")
	require.NoError(t, err)
	return account
}

func openTestLicense(t *testing.T, features ...string) {
	t.Helper()
	inserted, err := models.InsertOpenLicenseActivation(database.Conn(), &models.InstallationLicenseActivation{
		LicenseID:       uuid.New(),
		LicenseFeatures: features,
	})
	require.NoError(t, err)
	require.True(t, inserted)
}
