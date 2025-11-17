package authorization_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func TestMultiTenantEnforcer_Verify(t *testing.T) {
	r := support.Setup(t)

	userID := r.User.String()
	orgID := r.Organization.ID.String()

	allowed, err := authorization.Verify(userID, orgID, "canvas", "read")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestMultiTenantEnforcer_UpdateOrganizationPolicy(t *testing.T) {
	// r := support.Setup(t)

	// userID := r.User.String()
	// orgID := r.Organization.ID.String()

	// // Sanity check: org owner should initially have canvas read permission.
	// allowedBefore, err := authorization.Verify(userID, orgID, "canvas", "read")
	// require.NoError(t, err)
	// require.True(t, allowedBefore)

	// err = authorization.Update(orgID, func(tx *casbin.Transaction) error {
	// 	domain := fmt.Sprintf("%s:%s", models.DomainTypeOrganization, orgID)
	// 	role := fmt.Sprintf("role:%s", models.RoleOrgViewer)

	// 	oldPolicy := []string{role, domain, "canvas", "read"}
	// 	newPolicy := []string{role, domain, "canvas", "none"}

	// 	_, err := tx.RemovePolicy(oldPolicy...)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	_, err = tx.AddPolicy(newPolicy...)
	// 	return err
	// })
	// require.NoError(t, err)

	// // After update, org owner (who inherits org_viewer) should no longer have canvas read.
	// allowedAfter, err := authorization.Verify(userID, orgID, "canvas", "read")
	// require.NoError(t, err)
	// assert.False(t, allowedAfter)

	// // But other permissions (like create) should still be granted via org_admin.
	// allowedCreate, err := authorization.Verify(userID, orgID, "canvas", "create")
	// require.NoError(t, err)
	// assert.True(t, allowedCreate)
}

func TestMultiTenantEnforcer_OrganizationPolicyRecordCountsPerOrg(t *testing.T) {
	// r := support.Setup(t)

	// // Create a total of at least 10 organizations (including the one from Setup)
	// orgIDs := []string{r.Organization.ID.String()}
	// for i := 0; i < 9; i++ {
	// 	org := support.CreateOrganization(t, r, r.User)
	// 	orgIDs = append(orgIDs, org.ID.String())
	// }

	// db := database.Conn()

	// var firstCount int64
	// for i, orgID := range orgIDs {
	// 	domain := fmt.Sprintf("%s:%s", models.DomainTypeOrganization, orgID)
	// 	var count int64
	// 	require.NoError(t, db.Model(&gormadapter.CasbinRule{}).Where("v1 = ?", domain).Count(&count).Error)
	// 	require.Greater(t, count, int64(0))

	// 	if i == 0 {
	// 		firstCount = count
	// 	} else {
	// 		// Each organization should have the same number of Casbin
	// 		// rules stored for its domain (template policies + owner assignment).
	// 		assert.Equal(t, firstCount, count)
	// 	}
	// }
}

func TestMultiTenantEnforcer_Update_DoesNotChangeRecordCountOrOtherOrgs(t *testing.T) {
	// r := support.Setup(t)

	// // Create a total of at least 10 organizations (including the one from Setup)
	// orgIDs := []string{r.Organization.ID.String()}
	// for i := 0; i < 9; i++ {
	// 	org := support.CreateOrganization(t, r, r.User)
	// 	orgIDs = append(orgIDs, org.ID.String())
	// }

	// db := database.Conn()

	// // Capture record counts for all organizations before the update.
	// beforeCounts := make(map[string]int64)
	// for _, orgID := range orgIDs {
	// 	domain := fmt.Sprintf("%s:%s", models.DomainTypeOrganization, orgID)
	// 	var count int64
	// 	require.NoError(t, db.Model(&gormadapter.CasbinRule{}).Where("v1 = ?", domain).Count(&count).Error)
	// 	beforeCounts[domain] = count
	// }

	// targetOrgID := orgIDs[0]
	// targetDomain := fmt.Sprintf("%s:%s", models.DomainTypeOrganization, targetOrgID)

	// err := authorization.Update(targetOrgID, func(tx *casbin.Transaction) error {
	// 	role := fmt.Sprintf("role:%s", models.RoleOrgViewer)

	// 	oldPolicy := []string{role, targetDomain, "canvas", "read"}
	// 	newPolicy := []string{role, targetDomain, "canvas", "none"}

	// 	_, err := tx.RemovePolicy(oldPolicy...)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	_, err = tx.AddPolicy(newPolicy...)
	// 	return err
	// })
	// require.NoError(t, err)

	// // Capture record counts again after the update.
	// afterCounts := make(map[string]int64)
	// for _, orgID := range orgIDs {
	// 	domain := fmt.Sprintf("%s:%s", models.DomainTypeOrganization, orgID)
	// 	var count int64
	// 	require.NoError(t, db.Model(&gormadapter.CasbinRule{}).Where("v1 = ?", domain).Count(&count).Error)
	// 	afterCounts[domain] = count
	// }

	// // Update should not change how many records are stored for the target org,
	// // and must not affect other organizations' records.
	// for domain, before := range beforeCounts {
	// 	after := afterCounts[domain]
	// 	assert.Equal(t, before, after, "record count changed for domain %s", domain)
	// }
}
