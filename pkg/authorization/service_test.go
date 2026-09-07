package authorization_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__AuthService_BasicPermissions(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()
	canvasPath := "canvases"

	t.Run("user without roles has no permissions", func(t *testing.T) {
		allowedOrg, err := r.AuthService.CheckOrganizationPermission(context.Background(), uuid.NewString(), orgID, canvasPath, "read")
		require.NoError(t, err)
		assert.False(t, allowedOrg)
	})
}

func Test__AuthService_OrganizationPermissions(t *testing.T) {
	r := support.Setup(t)
	userID := r.User.String()
	orgID := r.Organization.ID.String()
	canvasPath := "canvases"
	memberPath := "members"
	orgPath := "org"

	t.Run("org admin has all permissions", func(t *testing.T) {
		err := r.AuthService.AssignRole(userID, models.RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		actions := []string{"read", "create", "update", "delete"}
		for _, action := range actions {
			allowed, err := r.AuthService.CheckOrganizationPermission(context.Background(), userID, orgID, canvasPath, action)
			require.NoError(t, err)
			assert.True(t, allowed, "Org admin should have %s permission for canvases", action)
		}

		allowed, err := r.AuthService.CheckOrganizationPermission(context.Background(), userID, orgID, memberPath, "create")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), userID, orgID, memberPath, "update")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), userID, orgID, memberPath, "delete")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), userID, orgID, orgPath, "update")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), userID, orgID, orgPath, "delete")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("org maintainer can edit automations but not members or org settings", func(t *testing.T) {
		maintainerID := uuid.New().String()
		err := r.AuthService.AssignRole(maintainerID, models.RoleOrgMaintainer, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		actions := []string{"read", "create", "update", "delete"}
		for _, action := range actions {
			allowed, err := r.AuthService.CheckOrganizationPermission(context.Background(), maintainerID, orgID, canvasPath, action)
			require.NoError(t, err)
			assert.True(t, allowed, "Org maintainer should have %s permission for canvases", action)
		}

		allowed, err := r.AuthService.CheckOrganizationPermission(context.Background(), maintainerID, orgID, "integrations", "create")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), maintainerID, orgID, "factories", "update")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), maintainerID, orgID, memberPath, "create")
		require.NoError(t, err)
		assert.False(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), maintainerID, orgID, orgPath, "update")
		require.NoError(t, err)
		assert.False(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), maintainerID, orgID, orgPath, "delete")
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("org operator can create tasks and cannot edit automations", func(t *testing.T) {
		operatorID := uuid.New().String()
		err := r.AuthService.AssignRole(operatorID, models.RoleOrgOperator, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		allowed, err := r.AuthService.CheckOrganizationPermission(context.Background(), operatorID, orgID, canvasPath, "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		actions := []string{"create", "update", "delete"}
		for _, action := range actions {
			allowed, err := r.AuthService.CheckOrganizationPermission(context.Background(), operatorID, orgID, canvasPath, action)
			require.NoError(t, err)
			assert.False(t, allowed, "Org operator should not have %s permission for canvases", action)
		}

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), operatorID, orgID, "work_orders", "create")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), operatorID, orgID, "work_orders", "update")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), operatorID, orgID, memberPath, "create")
		require.NoError(t, err)
		assert.False(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), operatorID, orgID, "notifications", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), operatorID, orgID, "notifications", "update")
		require.NoError(t, err)
		assert.True(t, allowed)
	})
}

func Test__AuthService_RoleManagement(t *testing.T) {
	r := support.Setup(t)
	userID := r.User.String()
	orgID := r.Organization.ID.String()
	canvasPath := "canvases"

	t.Run("assign and remove roles", func(t *testing.T) {
		err := r.AuthService.AssignRole(userID, models.RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		roles, err := r.AuthService.GetUserRolesForOrg(context.Background(), userID, orgID)
		require.NoError(t, err)
		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}
		require.True(t, flatRoles[models.RoleOrgAdmin])
		allowed, err := r.AuthService.CheckOrganizationPermission(context.Background(), userID, orgID, canvasPath, "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		err = r.AuthService.RemoveRole(userID, models.RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		roles, err = r.AuthService.GetUserRolesForOrg(context.Background(), userID, orgID)
		require.NoError(t, err)
		assert.NotContains(t, roles, models.RoleOrgAdmin)
		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), userID, orgID, canvasPath, "read")
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("invalid role assignment", func(t *testing.T) {
		err := r.AuthService.AssignRole(userID, "invalid_role", orgID, models.DomainTypeOrganization)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role")
	})
}

func Test__AuthService_GroupManagement(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()
	canvasPath := "canvases"

	t.Run("create and manage groups", func(t *testing.T) {
		groupName := "engineering-team"

		err := r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, groupName, models.RoleOrgAdmin, "Engineering Team", "Engineering Team")
		require.NoError(t, err)

		user1 := uuid.New().String()
		user2 := uuid.New().String()

		err = r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, user1, groupName)
		require.NoError(t, err)
		err = r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, user2, groupName)
		require.NoError(t, err)

		users, err := r.AuthService.GetGroupUsers(context.Background(), orgID, models.DomainTypeOrganization, groupName)
		require.NoError(t, err)
		assert.Contains(t, users, user1)
		assert.Contains(t, users, user2)

		allowed, err := r.AuthService.CheckOrganizationPermission(context.Background(), user1, orgID, canvasPath, "create")
		require.NoError(t, err)
		assert.True(t, allowed)

		err = r.AuthService.RemoveUserFromGroup(orgID, models.DomainTypeOrganization, user1, groupName)
		require.NoError(t, err)

		users, err = r.AuthService.GetGroupUsers(context.Background(), orgID, models.DomainTypeOrganization, groupName)
		require.NoError(t, err)
		assert.NotContains(t, users, user1)
		assert.Contains(t, users, user2)
	})

	t.Run("create group with invalid role", func(t *testing.T) {
		err := r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group", "invalid_role", "Test Group", "Test Group")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role")
	})

	t.Run("add user to non-existent group", func(t *testing.T) {
		userID := uuid.New().String()
		err := r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, userID, "non-existent-group")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("get groups and roles", func(t *testing.T) {
		err := r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "admins", models.RoleOrgAdmin, "Admins", "Admins")
		require.NoError(t, err)
		err = r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "operators", models.RoleOrgOperator, "Operators", "Operators")
		require.NoError(t, err)

		user1 := uuid.New().String()
		user2 := uuid.New().String()
		err = r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, user1, "admins")
		require.NoError(t, err)
		err = r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, user2, "operators")
		require.NoError(t, err)

		groups, err := r.AuthService.GetGroups(context.Background(), orgID, models.DomainTypeOrganization)
		require.NoError(t, err)
		assert.Contains(t, groups, "admins")
		assert.Contains(t, groups, "operators")

		role, err := r.AuthService.GetGroupRole(context.Background(), orgID, models.DomainTypeOrganization, "admins")
		require.NoError(t, err)
		assert.Equal(t, role, models.RoleOrgAdmin)

		role, err = r.AuthService.GetGroupRole(context.Background(), orgID, models.DomainTypeOrganization, "operators")
		require.NoError(t, err)
		assert.Equal(t, role, models.RoleOrgOperator)
	})
}

func Test__AuthService_RoleHierarchy(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()

	t.Run("org admin inherits maintainer and operator permissions", func(t *testing.T) {
		roles, err := r.AuthService.GetUserRolesForOrg(context.Background(), r.User.String(), orgID)
		require.NoError(t, err)

		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		require.True(t, flatRoles[models.RoleOrgAdmin])
		require.True(t, flatRoles[models.RoleOrgMaintainer])
		require.True(t, flatRoles[models.RoleOrgOperator])
	})
}

func Test__AuthService_DuplicateAssignments(t *testing.T) {
	r := support.Setup(t)
	userID := r.User.String()
	orgID := r.Organization.ID.String()

	t.Run("duplicate role assignment is idempotent", func(t *testing.T) {
		err := r.AuthService.AssignRole(userID, models.RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		err = r.AuthService.AssignRole(userID, models.RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		roles, err := r.AuthService.GetUserRolesForOrg(context.Background(), userID, orgID)
		require.NoError(t, err)

		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		require.True(t, flatRoles[models.RoleOrgAdmin])
	})

	t.Run("duplicate group creation fails", func(t *testing.T) {
		groupName := "duplicate-test-group"

		err := r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, groupName, models.RoleOrgOperator, "Duplicate Test Group", "This is a duplicate test group")
		require.NoError(t, err)

		err = r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, groupName, models.RoleOrgOperator, "Duplicate Test Group", "This is a duplicate test group")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func Test__AuthService_PermissionBoundaries(t *testing.T) {
	r := support.Setup(t)

	t.Run("org update and delete are admin permissions", func(t *testing.T) {
		orgID := r.Organization.ID.String()
		orgPath := "org"
		operatorID := uuid.New().String()
		maintainerID := uuid.New().String()

		err := r.AuthService.AssignRole(operatorID, models.RoleOrgOperator, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)
		err = r.AuthService.AssignRole(maintainerID, models.RoleOrgMaintainer, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		allowed, err := r.AuthService.CheckOrganizationPermission(context.Background(), operatorID, orgID, orgPath, "update")
		require.NoError(t, err)
		assert.False(t, allowed, "Operator should not have org update permission")

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), maintainerID, orgID, orgPath, "update")
		require.NoError(t, err)
		assert.False(t, allowed, "Maintainer should not have org update permission")

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), r.User.String(), orgID, orgPath, "update")
		require.NoError(t, err)
		assert.True(t, allowed, "Admin should have org update permission")

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), operatorID, orgID, orgPath, "delete")
		require.NoError(t, err)
		assert.False(t, allowed, "Operator should not have org delete permission")

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), maintainerID, orgID, orgPath, "delete")
		require.NoError(t, err)
		assert.False(t, allowed, "Maintainer should not have org delete permission")

		allowed, err = r.AuthService.CheckOrganizationPermission(context.Background(), r.User.String(), orgID, orgPath, "delete")
		require.NoError(t, err)
		assert.True(t, allowed, "Admin should have org delete permission")
	})
}

func Test__AuthService_GetRoleDefinition(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()

	t.Run("get organization role definition", func(t *testing.T) {
		operatorRole, err := r.AuthService.GetRoleDefinition(context.Background(), models.RoleOrgOperator, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, models.RoleOrgOperator, operatorRole.Name)
		assert.Equal(t, models.DomainTypeOrganization, operatorRole.DomainType)
		assert.NotEmpty(t, operatorRole.Description)
		assert.True(t, operatorRole.Readonly)
		assert.NotEmpty(t, operatorRole.Permissions)

		maintainerRole, err := r.AuthService.GetRoleDefinition(context.Background(), models.RoleOrgMaintainer, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, models.RoleOrgMaintainer, maintainerRole.Name)
		assert.Equal(t, models.DomainTypeOrganization, maintainerRole.DomainType)
		assert.NotEmpty(t, maintainerRole.Description)
		assert.True(t, maintainerRole.Readonly)
		assert.NotEmpty(t, maintainerRole.Permissions)

		adminRole, err := r.AuthService.GetRoleDefinition(context.Background(), models.RoleOrgAdmin, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, models.RoleOrgAdmin, adminRole.Name)
		assert.Equal(t, models.DomainTypeOrganization, adminRole.DomainType)
		assert.NotEmpty(t, adminRole.Description)
		assert.True(t, adminRole.Readonly)
		assert.NotEmpty(t, adminRole.Permissions)
	})

	t.Run("error cases", func(t *testing.T) {
		_, err := r.AuthService.GetRoleDefinition(context.Background(), "non_existent_role", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		_, err = r.AuthService.GetRoleDefinition(context.Background(), models.RoleOrgOperator, "invalid_domain", orgID)
		assert.Error(t, err)
	})

	t.Run("permissions are populated", func(t *testing.T) {
		role, err := r.AuthService.GetRoleDefinition(context.Background(), models.RoleOrgAdmin, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)

		for _, perm := range role.Permissions {
			assert.NotEmpty(t, perm.Resource)
			assert.NotEmpty(t, perm.Action)
			assert.Equal(t, models.DomainTypeOrganization, perm.DomainType)
		}
	})
}

func Test__AuthService_GetAllRoleDefinitions(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()

	t.Run("get all organization roles", func(t *testing.T) {
		roles, err := r.AuthService.GetAllRoleDefinitions(context.Background(), models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3)

		roleNames := make([]string, len(roles))
		for i, role := range roles {
			roleNames[i] = role.Name
		}

		assert.Contains(t, roleNames, models.RoleOrgOperator)
		assert.Contains(t, roleNames, models.RoleOrgMaintainer)
		assert.Contains(t, roleNames, models.RoleOrgAdmin)

		for _, role := range roles {
			assert.NotEmpty(t, role.Name)
			assert.Equal(t, models.DomainTypeOrganization, role.DomainType)
			assert.NotEmpty(t, role.Description)
			assert.True(t, role.Readonly)
			assert.NotEmpty(t, role.Permissions)
		}
	})

	t.Run("domain isolation", func(t *testing.T) {
		anotherOrg := support.CreateOrganization(t, r, r.User)

		roles1, err := r.AuthService.GetAllRoleDefinitions(context.Background(), models.DomainTypeOrganization, orgID)
		require.NoError(t, err)

		roles2, err := r.AuthService.GetAllRoleDefinitions(context.Background(), models.DomainTypeOrganization, anotherOrg.ID.String())
		require.NoError(t, err)
		assert.Equal(t, len(roles1), len(roles2))
	})

	t.Run("empty responses", func(t *testing.T) {
		definitions, _ := r.AuthService.GetAllRoleDefinitions(context.Background(), "invalid_domain", orgID)
		assert.Empty(t, definitions)

		definitions, _ = r.AuthService.GetAllRoleDefinitions(context.Background(), models.DomainTypeOrganization, "non-existent-org")
		assert.NotEmpty(t, definitions)
	})
}

func Test__AuthService_GetRolePermissions(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()

	t.Run("get organization role permissions", func(t *testing.T) {
		operatorPermissions, err := r.AuthService.GetRolePermissions(context.Background(), models.RoleOrgOperator, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.NotEmpty(t, operatorPermissions)

		hasNotificationRead := false
		hasNotificationUpdate := false
		hasWorkOrderCreate := false
		for _, perm := range operatorPermissions {
			assert.Equal(t, models.DomainTypeOrganization, perm.DomainType)
			switch {
			case perm.Resource == "notifications" && perm.Action == "update":
				hasNotificationUpdate = true
			case perm.Resource == "notifications" && perm.Action == "read":
				hasNotificationRead = true
			case perm.Resource == "work_orders" && perm.Action == "create":
				hasWorkOrderCreate = true
			case perm.Action != "read" && perm.Resource != "work_orders" && perm.Resource != "notifications":
				assert.Equal(t, "read", perm.Action, perm.Resource)
			}
		}
		assert.True(t, hasNotificationRead)
		assert.True(t, hasNotificationUpdate)
		assert.True(t, hasWorkOrderCreate)

		maintainerPermissions, err := r.AuthService.GetRolePermissions(context.Background(), models.RoleOrgMaintainer, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.NotEmpty(t, maintainerPermissions)
		assert.GreaterOrEqual(t, len(maintainerPermissions), len(operatorPermissions))

		adminPermissions, err := r.AuthService.GetRolePermissions(context.Background(), models.RoleOrgAdmin, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.NotEmpty(t, adminPermissions)
		assert.GreaterOrEqual(t, len(adminPermissions), len(maintainerPermissions))

		actions := make(map[string]bool)
		for _, perm := range adminPermissions {
			actions[perm.Action] = true
			assert.Equal(t, models.DomainTypeOrganization, perm.DomainType)
		}
		assert.True(t, actions["read"], "Admin should have read permissions")
		assert.True(t, actions["update"], "Admin should have update permissions")
		assert.True(t, actions["delete"], "Admin should have delete permissions")
	})

	t.Run("permissions include inheritance", func(t *testing.T) {
		operatorPermissions, err := r.AuthService.GetRolePermissions(context.Background(), models.RoleOrgOperator, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)

		maintainerPermissions, err := r.AuthService.GetRolePermissions(context.Background(), models.RoleOrgMaintainer, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)

		operatorPermMap := make(map[string]bool)
		for _, perm := range operatorPermissions {
			key := perm.Resource + ":" + perm.Action
			operatorPermMap[key] = true
		}

		maintainerPermMap := make(map[string]bool)
		for _, perm := range maintainerPermissions {
			key := perm.Resource + ":" + perm.Action
			maintainerPermMap[key] = true
		}

		for operatorPerm := range operatorPermMap {
			assert.True(t, maintainerPermMap[operatorPerm], "Maintainer should have operator permission: %s", operatorPerm)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		_, err := r.AuthService.GetRolePermissions(context.Background(), "non_existent_role", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)

		_, err = r.AuthService.GetRolePermissions(context.Background(), models.RoleOrgOperator, "invalid_domain", orgID)
		assert.Error(t, err)
	})
}

func Test__AuthService_GetRoleHierarchy(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()

	t.Run("get organization role hierarchy", func(t *testing.T) {
		operatorHierarchy, err := r.AuthService.GetRoleHierarchy(context.Background(), models.RoleOrgOperator, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Contains(t, operatorHierarchy, models.RoleOrgOperator)

		maintainerHierarchy, err := r.AuthService.GetRoleHierarchy(context.Background(), models.RoleOrgMaintainer, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Contains(t, maintainerHierarchy, models.RoleOrgMaintainer)

		adminHierarchy, err := r.AuthService.GetRoleHierarchy(context.Background(), models.RoleOrgAdmin, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Contains(t, adminHierarchy, models.RoleOrgAdmin)
		assert.GreaterOrEqual(t, len(adminHierarchy), len(maintainerHierarchy))
	})

	t.Run("hierarchy includes inheritance", func(t *testing.T) {
		adminHierarchy, err := r.AuthService.GetRoleHierarchy(context.Background(), models.RoleOrgAdmin, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(adminHierarchy), 1)

		maintainerHierarchy, err := r.AuthService.GetRoleHierarchy(context.Background(), models.RoleOrgMaintainer, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(maintainerHierarchy), len(adminHierarchy))
	})

	t.Run("hierarchy is unique", func(t *testing.T) {
		hierarchy, err := r.AuthService.GetRoleHierarchy(context.Background(), models.RoleOrgAdmin, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)

		seen := make(map[string]bool)
		for _, role := range hierarchy {
			assert.False(t, seen[role], "Role %s should not appear twice in hierarchy", role)
			seen[role] = true
		}
	})

	t.Run("error cases", func(t *testing.T) {
		_, err := r.AuthService.GetRoleHierarchy(context.Background(), "non_existent_role", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)

		_, err = r.AuthService.GetRoleHierarchy(context.Background(), models.RoleOrgOperator, "invalid_domain", orgID)
		assert.Error(t, err)
	})
}
