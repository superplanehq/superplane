package authorization_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestCreateRejectsLegacyResourceNames(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	orgID := r.Organization.ID.String()
	domain := "/org/" + orgID
	for _, kind := range []string{"role", "group"} {
		for _, representation := range []string{"metadata", "permission", "grouping subject", "grouping target", "builtin"} {
			if kind == "group" && (representation == "permission" || representation == "builtin") {
				continue
			}
			t.Run(kind+"/"+representation, func(t *testing.T) {
				name := uuid.NewString()
				identity := "/" + kind + "s/" + name
				switch representation {
				case "metadata":
					if kind == "role" {
						require.NoError(t, db.Create(&models.RoleMetadata{RoleName: name, DomainType: models.DomainTypeOrganization, DomainID: orgID, DisplayName: "Legacy"}).Error)
					} else {
						require.NoError(t, db.Create(&models.GroupMetadata{GroupName: name, DomainType: models.DomainTypeOrganization, DomainID: orgID, DisplayName: "Legacy"}).Error)
					}
				case "permission":
					require.NoError(t, db.Create(&gormadapter.CasbinRule{Ptype: "p", V0: identity, V1: domain, V2: "roles", V3: "create"}).Error)
				case "grouping subject":
					require.NoError(t, db.Create(&gormadapter.CasbinRule{Ptype: "g", V0: identity, V1: "/roles/org_operator", V2: domain}).Error)
				case "grouping target":
					require.NoError(t, db.Create(&gormadapter.CasbinRule{Ptype: "g", V0: "/users/" + uuid.NewString(), V1: identity, V2: domain}).Error)
				case "builtin":
					name = models.RoleOrgAdmin
					// Built-in names remain occupied without organization metadata.
					require.NoError(t, db.Where("role_name = ? AND domain_id = ?", name, orgID).Delete(&models.RoleMetadata{}).Error)
				}
				var beforeRoles, afterRoles []models.RoleMetadata
				var beforeGroups, afterGroups []models.GroupMetadata
				var beforePolicies, afterPolicies []gormadapter.CasbinRule
				require.NoError(t, db.Order("id").Find(&beforeRoles).Error)
				require.NoError(t, db.Order("id").Find(&beforeGroups).Error)
				require.NoError(t, db.Order("id").Find(&beforePolicies).Error)
				err := createTestResource(r.AuthService, kind, orgID, name, models.RoleOrgAdmin)
				require.ErrorIs(t, err, authorization.ErrResourceAlreadyExists)
				require.NoError(t, db.Order("id").Find(&afterRoles).Error)
				require.NoError(t, db.Order("id").Find(&afterGroups).Error)
				require.NoError(t, db.Order("id").Find(&afterPolicies).Error)
				assert.Equal(t, beforeRoles, afterRoles)
				assert.Equal(t, beforeGroups, afterGroups)
				assert.Equal(t, beforePolicies, afterPolicies)
				if representation != "builtin" {
					// Names are scoped to the exact organization and resource kind.
					require.NoError(t, createTestResource(r.AuthService, kind, uuid.NewString(), name, models.RoleOrgOperator))
					otherKind := "role"
					if kind == "role" {
						otherKind = "group"
					}
					require.NoError(t, createTestResource(r.AuthService, otherKind, orgID, name, models.RoleOrgOperator))
				}
			})
		}
	}
}

func createTestResource(service *authorization.AuthService, kind, orgID, name, inheritedRole string) error {
	if kind == "group" {
		return service.CreateGroup(orgID, models.DomainTypeOrganization, name, inheritedRole, inheritedRole, "Requested description")
	}
	return service.CreateCustomRole(orgID, &authorization.RoleDefinition{
		Name: name, DomainType: models.DomainTypeOrganization, DisplayName: inheritedRole, Description: "Requested description",
		Permissions:  []*authorization.Permission{{Resource: "roles", Action: "read"}},
		InheritsFrom: &authorization.RoleDefinition{Name: inheritedRole},
	})
}

func TestConcurrentCreationCommitsOnlyWinnerGrants(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	orgID := r.Organization.ID.String()
	first, err := authorization.NewAuthService()
	require.NoError(t, err)
	second, err := authorization.NewAuthService()
	require.NoError(t, err)
	services := []*authorization.AuthService{first, second}
	grants := []string{models.RoleOrgOperator, models.RoleOrgAdmin}
	for _, kind := range []string{"role", "group"} {
		for iteration := range 3 {
			t.Run(fmt.Sprintf("%s/%d", kind, iteration), func(t *testing.T) {
				name := uuid.NewString()
				// Hold the same database lock so both independent transactions contend.
				lock := db.Begin()
				require.NoError(t, lock.Error)
				defer lock.Rollback()
				require.NoError(t, models.LockRBACResourceName(lock, kind, models.DomainTypeOrganization, orgID, name))
				type result struct {
					index int
					err   error
				}
				results := make(chan result, 2)
				started := make(chan struct{}, 2)
				for index, service := range services {
					go func() {
						started <- struct{}{}
						results <- result{index, createTestResource(service, kind, orgID, name, grants[index])}
					}()
				}
				<-started
				<-started
				require.Eventually(t, func() bool {
					var waiting int64
					err := db.Raw("SELECT count(*) FROM pg_locks WHERE locktype = 'advisory' AND NOT granted AND database = (SELECT oid FROM pg_database WHERE datname = current_database())").Scan(&waiting).Error
					return err == nil && waiting == 2
				}, time.Second, 5*time.Millisecond, "both database sessions must contend for the name")
				require.NoError(t, lock.Commit().Error)
				winner := -1
				for range 2 {
					select {
					case result := <-results:
						if result.err == nil {
							require.Equal(t, -1, winner, "only one creation may succeed")
							winner = result.index
							continue
						}
						require.ErrorIs(t, result.err, authorization.ErrResourceAlreadyExists)
					case <-time.After(5 * time.Second):
						t.Fatal("concurrent creation did not finish")
					}
				}
				require.NotEqual(t, -1, winner)
				identity := "/" + kind + "s/" + name
				var policies []gormadapter.CasbinRule
				require.NoError(t, db.Where("v0 = ?", identity).Find(&policies).Error)
				expected := []gormadapter.CasbinRule{{Ptype: "g", V0: identity, V1: "/roles/" + grants[winner], V2: "/org/" + orgID}}
				if kind == "role" {
					expected = append(expected, gormadapter.CasbinRule{Ptype: "p", V0: identity, V1: "/org/" + orgID, V2: "roles", V3: "read"})
					var metadata []models.RoleMetadata
					require.NoError(t, db.Where("role_name = ? AND domain_id = ?", name, orgID).Find(&metadata).Error)
					require.Len(t, metadata, 1)
					assert.Equal(t, grants[winner], metadata[0].DisplayName)
					assert.Equal(t, "Requested description", metadata[0].Description)
				} else {
					var metadata []models.GroupMetadata
					require.NoError(t, db.Where("group_name = ? AND domain_id = ?", name, orgID).Find(&metadata).Error)
					require.Len(t, metadata, 1)
					assert.Equal(t, grants[winner], metadata[0].DisplayName)
					assert.Equal(t, "Requested description", metadata[0].Description)
				}
				for index := range policies {
					policies[index].ID = 0
				}
				assert.ElementsMatch(t, expected, policies)
			})
		}
	}
}

func TestCreationRollsBackMetadataAndPoliciesOnStorageFailure(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	orgID := r.Organization.ID.String()
	var before, after []gormadapter.CasbinRule
	require.NoError(t, db.Order("id").Find(&before).Error)
	// Metadata permits 255 bytes, but the policy identity permits only 100.
	groupName := strings.Repeat("g", 101)
	err := createTestResource(r.AuthService, "group", orgID, groupName, models.RoleOrgAdmin)
	require.Error(t, err)
	assert.NotErrorIs(t, err, authorization.ErrResourceAlreadyExists)
	roleName := "rollback-role"
	err = r.AuthService.CreateCustomRole(orgID, &authorization.RoleDefinition{
		Name: roleName, DomainType: models.DomainTypeOrganization,
		Permissions: []*authorization.Permission{
			{Resource: "roles", Action: "read"},
			{Resource: strings.Repeat("r", 101), Action: "read"},
		},
	})
	require.Error(t, err)
	assert.NotErrorIs(t, err, authorization.ErrResourceAlreadyExists)
	var roles []models.RoleMetadata
	var groups []models.GroupMetadata
	require.NoError(t, db.Where("role_name = ?", roleName).Find(&roles).Error)
	require.NoError(t, db.Where("group_name = ?", groupName).Find(&groups).Error)
	assert.Empty(t, roles)
	assert.Empty(t, groups)
	require.NoError(t, db.Order("id").Find(&after).Error)
	assert.Equal(t, before, after)
	// A fresh permission view must not retain the rolled-back role policy.
	err = r.AuthService.AssignRole(uuid.NewString(), roleName, orgID, models.DomainTypeOrganization)
	assert.Error(t, err)
}
