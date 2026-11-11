package enterprise

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestPlanLicenseInstall(t *testing.T) {
	licenseID := uuid.New()
	otherID := uuid.New()
	orgID := uuid.New()
	otherOrgID := uuid.New()
	license := &licensing.License{ID: licenseID, Features: []licensing.Feature{licensing.FeatureCustomRoles, licensing.FeatureGroups}}
	groups := models.OrganizationEnterpriseFeature{OrganizationID: orgID, FeatureKey: "groups"}
	roles := models.OrganizationEnterpriseFeature{OrganizationID: orgID, FeatureKey: "custom_roles"}
	otherGroups := models.OrganizationEnterpriseFeature{OrganizationID: otherOrgID, FeatureKey: "groups"}

	t.Run("first license for one organization grants every feature", func(t *testing.T) {
		plan := PlanLicenseInstall(nil, license, nil, 1, orgID)
		assert.Equal(t, LicenseEventActivation, plan.Event)
		assert.Empty(t, plan.Removed)
		assert.Equal(t, orgID, plan.AutoGrantOrganizationID)
	})

	t.Run("first license for several organizations grants nothing", func(t *testing.T) {
		plan := PlanLicenseInstall(nil, license, nil, 3, uuid.Nil)
		assert.Equal(t, LicenseEventActivation, plan.Event)
		assert.Empty(t, plan.Removed)
		assert.Equal(t, uuid.Nil, plan.AutoGrantOrganizationID)
	})

	t.Run("the same license does not change grants", func(t *testing.T) {
		open := &models.InstallationLicenseActivation{LicenseID: licenseID}
		plan := PlanLicenseInstall(open, license, []models.OrganizationEnterpriseFeature{{
			OrganizationID: orgID,
			FeatureKey:     "audit_logs",
		}}, 1, orgID)
		assert.Equal(t, LicenseEventNone, plan.Event)
		assert.Empty(t, plan.Removed)
		assert.Equal(t, uuid.Nil, plan.AutoGrantOrganizationID)
	})

	t.Run("replacement removes only features the new license does not contain", func(t *testing.T) {
		open := &models.InstallationLicenseActivation{LicenseID: otherID}
		narrow := &licensing.License{ID: licenseID, Features: []licensing.Feature{licensing.FeatureCustomRoles}}
		plan := PlanLicenseInstall(open, narrow, []models.OrganizationEnterpriseFeature{roles, groups, otherGroups}, 3, uuid.Nil)
		assert.Equal(t, LicenseEventReplacement, plan.Event)
		assert.ElementsMatch(t, []PlannedGrantRemoval{
			{OrganizationID: orgID, FeatureKey: "groups"},
			{OrganizationID: otherOrgID, FeatureKey: "groups"},
		}, plan.Removed)
		assert.Equal(t, uuid.Nil, plan.AutoGrantOrganizationID)
	})

	t.Run("replacement grants a new feature only to one organization", func(t *testing.T) {
		open := &models.InstallationLicenseActivation{LicenseID: otherID}
		plan := PlanLicenseInstall(open, license, []models.OrganizationEnterpriseFeature{roles}, 1, orgID)
		assert.Equal(t, LicenseEventReplacement, plan.Event)
		assert.Empty(t, plan.Removed)
		assert.Equal(t, orgID, plan.AutoGrantOrganizationID)
	})

	t.Run("one organization that already has every feature is not marked", func(t *testing.T) {
		open := &models.InstallationLicenseActivation{LicenseID: otherID}
		plan := PlanLicenseInstall(open, license, []models.OrganizationEnterpriseFeature{roles, groups}, 1, orgID)
		assert.Equal(t, LicenseEventReplacement, plan.Event)
		assert.Empty(t, plan.Removed)
		assert.Equal(t, uuid.Nil, plan.AutoGrantOrganizationID)
	})
}

func TestPlanLicenseRemoval(t *testing.T) {
	first := uuid.New()
	second := uuid.New()
	removed := PlanLicenseRemoval([]models.OrganizationEnterpriseFeature{
		{OrganizationID: second, FeatureKey: "groups"},
		{OrganizationID: first, FeatureKey: "custom_roles"},
		{OrganizationID: first, FeatureKey: "groups"},
	})
	assert.ElementsMatch(t, []PlannedGrantRemoval{
		{OrganizationID: first, FeatureKey: "custom_roles"},
		{OrganizationID: first, FeatureKey: "groups"},
		{OrganizationID: second, FeatureKey: "groups"},
	}, removed)
}

func TestDescribeAppliedInstall(t *testing.T) {
	first := &models.InstallationLicenseActivation{ID: 1, LicenseID: uuid.New()}
	second := &models.InstallationLicenseActivation{
		ID:          2,
		LicenseID:   uuid.New(),
		AutoGranted: true,
		RemovedGrants: []models.RemovedEnterpriseGrant{{
			OrganizationID: uuid.New(),
			FeatureKey:     "groups",
			RemovedBy:      models.EnterpriseFeatureRemovedByActivation,
		}},
	}

	assert.Equal(t, LicenseEventNone, DescribeAppliedInstall(first, first).Event)
	assert.Equal(t, LicenseEventNone, DescribeAppliedInstall(nil, nil).Event)

	applied := DescribeAppliedInstall(nil, second)
	assert.Equal(t, LicenseEventActivation, applied.Event)
	assert.True(t, applied.AutoGranted)
	assert.Equal(t, []models.RemovedEnterpriseGrant(second.RemovedGrants), applied.Removed)

	replaced := DescribeAppliedInstall(first, second)
	assert.Equal(t, LicenseEventReplacement, replaced.Event)
	assert.Equal(t, []models.RemovedEnterpriseGrant(second.RemovedGrants), replaced.Removed)
}
