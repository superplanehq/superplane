package authorization_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__AuthService_BasicPermissions(t *testing.T) {
	r := support.Setup(t)
	userID := r.User.String()
	canvasID := r.Canvas.ID.String()
	orgID := r.Organization.ID.String()

	t.Run("user without roles has no permissions", func(t *testing.T) {
		allowedOrg, err := r.AuthService.CheckOrganizationPermission(uuid.NewString(), orgID, "canvas", "read")
		require.NoError(t, err)
		assert.False(t, allowedOrg)

		allowedCanvas, err := r.AuthService.CheckCanvasPermission(uuid.NewString(), canvasID, "stage", "read")
		require.NoError(t, err)
		assert.False(t, allowedCanvas)
	})

	t.Run("canvas owner has all permissions", func(t *testing.T) {
		roles, err := r.AuthService.GetUserRolesForCanvas(userID, canvasID)
		require.NoError(t, err)
		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}
		require.True(t, flatRoles[models.RoleCanvasOwner])
		require.True(t, flatRoles[models.RoleCanvasAdmin])
		require.True(t, flatRoles[models.RoleCanvasViewer])

		// Test viewer permissions (inherited)
		allowed, err := r.AuthService.CheckCanvasPermission(userID, canvasID, "eventsource", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckCanvasPermission(userID, canvasID, "stage", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckCanvasPermission(userID, canvasID, "stageevent", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Test admin permissions (inherited)
		resources := []string{"eventsource", "stage"}
		actions := []string{"create", "update", "delete"}
		for _, resource := range resources {
			for _, action := range actions {
				allowed, err := r.AuthService.CheckCanvasPermission(userID, canvasID, resource, action)
				require.NoError(t, err)
				assert.True(t, allowed, "Canvas owner should have %s permission for %s", action, resource)
			}
		}

		// Test stageevent approve permission
		allowed, err = r.AuthService.CheckCanvasPermission(userID, canvasID, "stageevent", "approve")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Test member permissions
		allowed, err = r.AuthService.CheckCanvasPermission(userID, canvasID, "member", "invite")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckCanvasPermission(userID, canvasID, "member", "remove")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("canvas viewer has only read permissions", func(t *testing.T) {
		viewerID := uuid.New().String()
		err := r.AuthService.AssignRole(viewerID, models.RoleCanvasViewer, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		// Should have read permissions
		allowed, err := r.AuthService.CheckCanvasPermission(viewerID, canvasID, "eventsource", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckCanvasPermission(viewerID, canvasID, "stage", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckCanvasPermission(viewerID, canvasID, "stageevent", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should not have write permissions
		allowed, err = r.AuthService.CheckCanvasPermission(viewerID, canvasID, "stage", "create")
		require.NoError(t, err)
		assert.False(t, allowed)

		allowed, err = r.AuthService.CheckCanvasPermission(viewerID, canvasID, "stage", "update")
		require.NoError(t, err)
		assert.False(t, allowed)

		// Should not have approve permission
		allowed, err = r.AuthService.CheckCanvasPermission(viewerID, canvasID, "stageevent", "approve")
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("canvas admin has read and write permissions", func(t *testing.T) {
		adminID := uuid.New().String()
		err := r.AuthService.AssignRole(adminID, models.RoleCanvasAdmin, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		// Should have read permissions (inherited from viewer)
		allowed, err := r.AuthService.CheckCanvasPermission(adminID, canvasID, "stage", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should have create/update/delete permissions
		resources := []string{"eventsource", "stage"}
		actions := []string{"create", "update", "delete"}
		for _, resource := range resources {
			for _, action := range actions {
				allowed, err := r.AuthService.CheckCanvasPermission(adminID, canvasID, resource, action)
				require.NoError(t, err)
				assert.True(t, allowed, "Canvas admin should have %s permission for %s", action, resource)
			}
		}

		// Should have approve permission for stageevent
		allowed, err = r.AuthService.CheckCanvasPermission(adminID, canvasID, "stageevent", "approve")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should have member invite permission
		allowed, err = r.AuthService.CheckCanvasPermission(adminID, canvasID, "member", "invite")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should not have member remove permission (owner only)
		allowed, err = r.AuthService.CheckCanvasPermission(adminID, canvasID, "member", "remove")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__AuthService_OrganizationPermissions(t *testing.T) {
	r := support.Setup(t)
	userID := r.User.String()
	orgID := r.Organization.ID.String()

	t.Run("org owner has all permissions", func(t *testing.T) {
		err := r.AuthService.AssignRole(userID, models.RoleOrgOwner, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Should have all canvas permissions (inherited from admin)
		actions := []string{"read", "create", "update", "delete"}
		for _, action := range actions {
			allowed, err := r.AuthService.CheckOrganizationPermission(userID, orgID, "canvas", action)
			require.NoError(t, err)
			assert.True(t, allowed, "Org owner should have %s permission for canvas", action)
		}

		// Should have user management permissions (inherited from admin)
		allowed, err := r.AuthService.CheckOrganizationPermission(userID, orgID, "user", "invite")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(userID, orgID, "user", "remove")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should have org management permissions (owner only)
		allowed, err = r.AuthService.CheckOrganizationPermission(userID, orgID, models.DomainTypeOrganization, "update")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(userID, orgID, models.DomainTypeOrganization, "delete")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("org admin has limited permissions", func(t *testing.T) {
		adminID := uuid.New().String()
		err := r.AuthService.AssignRole(adminID, models.RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Should have canvas management permissions
		actions := []string{"read", "create", "update", "delete"}
		for _, action := range actions {
			allowed, err := r.AuthService.CheckOrganizationPermission(adminID, orgID, "canvas", action)
			require.NoError(t, err)
			assert.True(t, allowed, "Org admin should have %s permission for canvas", action)
		}

		// Should have user management permissions
		allowed, err := r.AuthService.CheckOrganizationPermission(adminID, orgID, "user", "invite")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(adminID, orgID, "user", "remove")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should not have org management permissions
		allowed, err = r.AuthService.CheckOrganizationPermission(adminID, orgID, models.DomainTypeOrganization, "update")
		require.NoError(t, err)
		assert.False(t, allowed)

		allowed, err = r.AuthService.CheckOrganizationPermission(adminID, orgID, models.DomainTypeOrganization, "delete")
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("org viewer has only read permissions", func(t *testing.T) {
		viewerID := uuid.New().String()
		err := r.AuthService.AssignRole(viewerID, models.RoleOrgViewer, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Should have read permission
		allowed, err := r.AuthService.CheckOrganizationPermission(viewerID, orgID, "canvas", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should not have create/update/delete permissions
		actions := []string{"create", "update", "delete"}
		for _, action := range actions {
			allowed, err := r.AuthService.CheckOrganizationPermission(viewerID, orgID, "canvas", action)
			require.NoError(t, err)
			assert.False(t, allowed, "Org viewer should not have %s permission for canvas", action)
		}

		// Should not have user management permissions
		allowed, err = r.AuthService.CheckOrganizationPermission(viewerID, orgID, "user", "invite")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__AuthService_RoleManagement(t *testing.T) {
	r := support.Setup(t)
	userID := r.User.String()
	orgID := r.Organization.ID.String()
	canvasID := r.Canvas.ID.String()

	t.Run("assign and remove roles", func(t *testing.T) {
		// Assign role
		err := r.AuthService.AssignRole(userID, models.RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Verify role assignment
		roles, err := r.AuthService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)
		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}
		require.True(t, flatRoles[models.RoleOrgAdmin])
		// Check permissions
		allowed, err := r.AuthService.CheckOrganizationPermission(userID, orgID, "canvas", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Remove role
		err = r.AuthService.RemoveRole(userID, models.RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Verify role removal
		roles, err = r.AuthService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)
		assert.NotContains(t, roles, models.RoleOrgAdmin)
		// Check permissions
		allowed, err = r.AuthService.CheckOrganizationPermission(userID, orgID, "canvas", "read")
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("get users for role", func(t *testing.T) {
		user1 := uuid.New().String()
		user2 := uuid.New().String()

		err := r.AuthService.AssignRole(user1, models.RoleCanvasViewer, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)
		err = r.AuthService.AssignRole(user2, models.RoleCanvasViewer, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		users, err := r.AuthService.GetCanvasUsersForRole(models.RoleCanvasViewer, canvasID)
		require.NoError(t, err)
		assert.Contains(t, users, user1)
		assert.Contains(t, users, user2)
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

		users, err := r.AuthService.GetGroupUsers(orgID, models.DomainTypeOrganization, groupName)
		require.NoError(t, err)
		assert.Contains(t, users, user1)
		assert.Contains(t, users, user2)

		allowed, err := r.AuthService.CheckOrganizationPermission(user1, orgID, "canvas", "create")
		require.NoError(t, err)
		assert.True(t, allowed)

		err = r.AuthService.RemoveUserFromGroup(orgID, models.DomainTypeOrganization, user1, groupName)
		require.NoError(t, err)

		users, err = r.AuthService.GetGroupUsers(orgID, models.DomainTypeOrganization, groupName)
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
		err = r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "viewers", models.RoleOrgViewer, "Viewers", "Viewers")
		require.NoError(t, err)

		user1 := uuid.New().String()
		user2 := uuid.New().String()
		err = r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, user1, "admins")
		require.NoError(t, err)
		err = r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, user2, "viewers")
		require.NoError(t, err)

		groups, err := r.AuthService.GetGroups(orgID, models.DomainTypeOrganization)
		require.NoError(t, err)
		assert.Contains(t, groups, "admins")
		assert.Contains(t, groups, "viewers")

		role, err := r.AuthService.GetGroupRole(orgID, models.DomainTypeOrganization, "admins")
		require.NoError(t, err)
		assert.Equal(t, role, models.RoleOrgAdmin)

		role, err = r.AuthService.GetGroupRole(orgID, models.DomainTypeOrganization, "viewers")
		require.NoError(t, err)
		assert.Equal(t, role, models.RoleOrgViewer)
	})
}

func Test__AuthService_AccessibleResources(t *testing.T) {
	r := support.Setup(t)
	canvas1 := r.Canvas
	org2 := support.CreateOrganization(t, r, r.User)
	canvas2 := support.CreateCanvas(t, r, org2.ID, r.User)

	t.Run("get accessible canvases", func(t *testing.T) {
		canvases, err := r.AuthService.GetAccessibleCanvasesForUser(r.User.String())
		require.NoError(t, err)
		assert.Contains(t, canvases, canvas1.ID.String())
		assert.Contains(t, canvases, canvas2.ID.String())
	})
}

func Test__AuthService_CreateOrganizationOwner(t *testing.T) {
	r := support.Setup(t)
	userID := r.User.String()
	orgID := r.Organization.ID.String()

	t.Run("create organization owner", func(t *testing.T) {
		err := r.AuthService.CreateOrganizationOwner(userID, orgID)
		require.NoError(t, err)

		// Verify owner permissions
		allowed, err := r.AuthService.CheckOrganizationPermission(userID, orgID, models.DomainTypeOrganization, "update")
		require.NoError(t, err)
		assert.True(t, allowed)

		roles, err := r.AuthService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)
		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		require.True(t, flatRoles[models.RoleOrgOwner])
	})
}

func Test__AuthService_RoleHierarchy(t *testing.T) {
	r := support.Setup(t)
	userID := r.User.String()
	canvasID := r.Canvas.ID.String()
	orgID := r.Organization.ID.String()

	t.Run("canvas owner inherits admin and viewer permissions", func(t *testing.T) {
		err := r.AuthService.AssignRole(userID, models.RoleCanvasOwner, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		// Get implicit roles (should include inherited roles)
		roles, err := r.AuthService.GetUserRolesForCanvas(userID, canvasID)
		require.NoError(t, err)

		// Should have all three roles due to hierarchy
		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		require.True(t, flatRoles[models.RoleCanvasOwner])
		require.True(t, flatRoles[models.RoleCanvasAdmin])
		require.True(t, flatRoles[models.RoleCanvasViewer])
	})

	t.Run("canvas admin inherits viewer permissions", func(t *testing.T) {
		adminID := uuid.New().String()
		err := r.AuthService.AssignRole(adminID, models.RoleCanvasAdmin, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		roles, err := r.AuthService.GetUserRolesForCanvas(adminID, canvasID)
		require.NoError(t, err)

		// Should have admin and viewer roles
		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		require.True(t, flatRoles[models.RoleCanvasAdmin])
		require.True(t, flatRoles[models.RoleCanvasViewer])
		require.False(t, flatRoles[models.RoleCanvasOwner])
	})

	t.Run("org owner inherits admin and viewer permissions", func(t *testing.T) {
		roles, err := r.AuthService.GetUserRolesForOrg(r.User.String(), orgID)
		require.NoError(t, err)

		// Should have all three roles due to hierarchy
		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		require.True(t, flatRoles[models.RoleOrgOwner])
		require.True(t, flatRoles[models.RoleOrgAdmin])
		require.True(t, flatRoles[models.RoleOrgViewer])
	})
}

func Test__AuthService_DuplicateAssignments(t *testing.T) {
	r := support.Setup(t)
	userID := r.User.String()
	orgID := r.Organization.ID.String()

	t.Run("duplicate role assignment is idempotent", func(t *testing.T) {
		// First assignment
		err := r.AuthService.AssignRole(userID, models.RoleOrgOwner, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Duplicate assignment should not error
		err = r.AuthService.AssignRole(userID, models.RoleOrgOwner, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Should still have the role only once
		roles, err := r.AuthService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)

		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		require.True(t, flatRoles[models.RoleOrgAdmin])
	})

	t.Run("duplicate group creation fails", func(t *testing.T) {
		groupName := "duplicate-test-group"

		err := r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, groupName, models.RoleOrgViewer, "Duplicate Test Group", "This is a duplicate test group")
		require.NoError(t, err)

		err = r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, groupName, models.RoleOrgViewer, "Duplicate Test Group", "This is a duplicate test group")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func Test__AuthService_CrossDomainPermissions(t *testing.T) {
	r := support.Setup(t)

	t.Run("org role does not grant canvas permissions", func(t *testing.T) {
		userID := r.User.String()
		orgID := uuid.New().String()
		canvasID := uuid.New().String()

		err := r.AuthService.SetupOrganizationRoles(orgID)
		require.NoError(t, err)
		err = r.AuthService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		// Assign org owner role
		err = r.AuthService.AssignRole(userID, models.RoleOrgOwner, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Should not have canvas permissions
		allowed, err := r.AuthService.CheckCanvasPermission(userID, canvasID, "stage", "read")
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("canvas role does not grant org permissions", func(t *testing.T) {
		userID := r.User.String()
		canvasID := uuid.New().String()
		orgID := uuid.New().String()

		err := r.AuthService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)
		err = r.AuthService.SetupOrganizationRoles(orgID)
		require.NoError(t, err)

		// Assign canvas owner role
		err = r.AuthService.AssignRole(userID, models.RoleCanvasOwner, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		// Should not have org permissions
		allowed, err := r.AuthService.CheckOrganizationPermission(userID, orgID, "canvas", "read")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__AuthService_PermissionBoundaries(t *testing.T) {
	r := support.Setup(t)
	canvasID := r.Canvas.ID.String()

	t.Run("member remove is owner-only permission", func(t *testing.T) {
		viewerID := uuid.New().String()
		adminID := uuid.New().String()

		// Assign roles
		err := r.AuthService.AssignRole(viewerID, models.RoleCanvasViewer, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)
		err = r.AuthService.AssignRole(adminID, models.RoleCanvasAdmin, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		// Viewer should not have member remove permission
		allowed, err := r.AuthService.CheckCanvasPermission(viewerID, canvasID, "member", "remove")
		require.NoError(t, err)
		assert.False(t, allowed)

		// Admin should not have member remove permission
		allowed, err = r.AuthService.CheckCanvasPermission(adminID, canvasID, "member", "remove")
		require.NoError(t, err)
		assert.False(t, allowed)

		// Owner should have member remove permission
		allowed, err = r.AuthService.CheckCanvasPermission(r.User.String(), canvasID, "member", "remove")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("org update and delete are owner-only permissions", func(t *testing.T) {
		orgID := r.Organization.ID.String()
		viewerID := uuid.New().String()
		adminID := uuid.New().String()

		// Assign roles
		err := r.AuthService.AssignRole(viewerID, models.RoleOrgViewer, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)
		err = r.AuthService.AssignRole(adminID, models.RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Check org update permission
		allowed, err := r.AuthService.CheckOrganizationPermission(viewerID, orgID, models.DomainTypeOrganization, "update")
		require.NoError(t, err)
		assert.False(t, allowed, "Viewer should not have org update permission")

		allowed, err = r.AuthService.CheckOrganizationPermission(adminID, orgID, models.DomainTypeOrganization, "update")
		require.NoError(t, err)
		assert.False(t, allowed, "Admin should not have org update permission")

		allowed, err = r.AuthService.CheckOrganizationPermission(r.User.String(), orgID, models.DomainTypeOrganization, "update")
		require.NoError(t, err)
		assert.True(t, allowed, "Owner should have org update permission")

		// Check org delete permission
		allowed, err = r.AuthService.CheckOrganizationPermission(viewerID, orgID, models.DomainTypeOrganization, "delete")
		require.NoError(t, err)
		assert.False(t, allowed, "Viewer should not have org delete permission")

		allowed, err = r.AuthService.CheckOrganizationPermission(adminID, orgID, models.DomainTypeOrganization, "delete")
		require.NoError(t, err)
		assert.False(t, allowed, "Admin should not have org delete permission")

		allowed, err = r.AuthService.CheckOrganizationPermission(r.User.String(), orgID, models.DomainTypeOrganization, "delete")
		require.NoError(t, err)
		assert.True(t, allowed, "Owner should have org delete permission")
	})
}

func Test__AuthService_GetRoleDefinition(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()
	canvasID := r.Canvas.ID.String()

	t.Run("get organization role definition", func(t *testing.T) {
		viewerRole, err := r.AuthService.GetRoleDefinition(models.RoleOrgViewer, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, models.RoleOrgViewer, viewerRole.Name)
		assert.Equal(t, models.DomainTypeOrganization, viewerRole.DomainType)
		assert.NotEmpty(t, viewerRole.Description)
		assert.True(t, viewerRole.Readonly)
		assert.NotEmpty(t, viewerRole.Permissions)

		// Test org admin role
		adminRole, err := r.AuthService.GetRoleDefinition(models.RoleOrgAdmin, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, models.RoleOrgAdmin, adminRole.Name)
		assert.Equal(t, models.DomainTypeOrganization, adminRole.DomainType)
		assert.NotEmpty(t, adminRole.Description)
		assert.True(t, adminRole.Readonly)
		assert.NotEmpty(t, adminRole.Permissions)

		// Test org owner role
		ownerRole, err := r.AuthService.GetRoleDefinition(models.RoleOrgOwner, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, models.RoleOrgOwner, ownerRole.Name)
		assert.Equal(t, models.DomainTypeOrganization, ownerRole.DomainType)
		assert.NotEmpty(t, ownerRole.Description)
		assert.True(t, ownerRole.Readonly)
		assert.NotEmpty(t, ownerRole.Permissions)
	})

	t.Run("get canvas role definition", func(t *testing.T) {
		// Test canvas viewer role
		viewerRole, err := r.AuthService.GetRoleDefinition(models.RoleCanvasViewer, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.Equal(t, models.RoleCanvasViewer, viewerRole.Name)
		assert.Equal(t, models.DomainTypeCanvas, viewerRole.DomainType)
		assert.NotEmpty(t, viewerRole.Description)
		assert.True(t, viewerRole.Readonly)
		assert.NotEmpty(t, viewerRole.Permissions)

		// Test canvas admin role
		adminRole, err := r.AuthService.GetRoleDefinition(models.RoleCanvasAdmin, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.Equal(t, models.RoleCanvasAdmin, adminRole.Name)
		assert.Equal(t, models.DomainTypeCanvas, adminRole.DomainType)
		assert.NotEmpty(t, adminRole.Description)
		assert.True(t, adminRole.Readonly)
		assert.NotEmpty(t, adminRole.Permissions)

		// Test canvas owner role
		ownerRole, err := r.AuthService.GetRoleDefinition(models.RoleCanvasOwner, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.Equal(t, models.RoleCanvasOwner, ownerRole.Name)
		assert.Equal(t, models.DomainTypeCanvas, ownerRole.DomainType)
		assert.NotEmpty(t, ownerRole.Description)
		assert.True(t, ownerRole.Readonly)
		assert.NotEmpty(t, ownerRole.Permissions)
	})

	t.Run("error cases", func(t *testing.T) {
		// Test non-existent role
		_, err := r.AuthService.GetRoleDefinition("non_existent_role", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test non-existent domain
		_, err = r.AuthService.GetRoleDefinition(models.RoleOrgViewer, models.DomainTypeOrganization, "non-existent-org")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test invalid domain type
		_, err = r.AuthService.GetRoleDefinition(models.RoleOrgViewer, "invalid_domain", orgID)
		assert.Error(t, err)
	})

	t.Run("permissions are populated", func(t *testing.T) {
		role, err := r.AuthService.GetRoleDefinition(models.RoleOrgAdmin, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)

		// Check that permissions have all required fields
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
	canvasID := r.Canvas.ID.String()

	t.Run("get all organization roles", func(t *testing.T) {
		roles, err := r.AuthService.GetAllRoleDefinitions(models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3) // Should have at least viewer, admin, owner

		// Extract role names
		roleNames := make([]string, len(roles))
		for i, role := range roles {
			roleNames[i] = role.Name
		}

		// Check that we have the expected roles
		assert.Contains(t, roleNames, models.RoleOrgViewer)
		assert.Contains(t, roleNames, models.RoleOrgAdmin)
		assert.Contains(t, roleNames, models.RoleOrgOwner)

		// Check that all roles have required fields
		for _, role := range roles {
			assert.NotEmpty(t, role.Name)
			assert.Equal(t, models.DomainTypeOrganization, role.DomainType)
			assert.NotEmpty(t, role.Description)
			assert.True(t, role.Readonly)
			assert.NotEmpty(t, role.Permissions)
		}
	})

	t.Run("get all canvas roles", func(t *testing.T) {
		roles, err := r.AuthService.GetAllRoleDefinitions(models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3) // Should have at least viewer, admin, owner

		// Extract role names
		roleNames := make([]string, len(roles))
		for i, role := range roles {
			roleNames[i] = role.Name
		}

		// Check that we have the expected roles
		assert.Contains(t, roleNames, models.RoleCanvasViewer)
		assert.Contains(t, roleNames, models.RoleCanvasAdmin)
		assert.Contains(t, roleNames, models.RoleCanvasOwner)

		// Check that all roles have required fields
		for _, role := range roles {
			assert.NotEmpty(t, role.Name)
			assert.Equal(t, models.DomainTypeCanvas, role.DomainType)
			assert.NotEmpty(t, role.Description)
			assert.True(t, role.Readonly)
			assert.NotEmpty(t, role.Permissions)
		}
	})

	t.Run("domain isolation", func(t *testing.T) {
		// Create another organization
		anotherOrg := support.CreateOrganization(t, r, r.User)

		// Both should have the same number of roles
		roles1, err := r.AuthService.GetAllRoleDefinitions(models.DomainTypeOrganization, orgID)
		require.NoError(t, err)

		roles2, err := r.AuthService.GetAllRoleDefinitions(models.DomainTypeOrganization, anotherOrg.ID.String())
		require.NoError(t, err)
		assert.Equal(t, len(roles1), len(roles2))
	})

	t.Run("empty responses", func(t *testing.T) {
		// Test invalid domain type
		definitions, _ := r.AuthService.GetAllRoleDefinitions("invalid_domain", orgID)
		assert.Empty(t, definitions)

		// Test non-existent domain
		definitions, _ = r.AuthService.GetAllRoleDefinitions(models.DomainTypeOrganization, "non-existent-org")
		assert.Empty(t, definitions)
	})
}

func Test__AuthService_GetRolePermissions(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()
	canvasID := r.Canvas.ID.String()

	t.Run("get organization role permissions", func(t *testing.T) {
		// Test org viewer permissions
		viewerPermissions, err := r.AuthService.GetRolePermissions(models.RoleOrgViewer, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.NotEmpty(t, viewerPermissions)

		// All permissions should be read-only
		for _, perm := range viewerPermissions {
			assert.Equal(t, "read", perm.Action)
			assert.Equal(t, models.DomainTypeOrganization, perm.DomainType)
		}

		// Test org admin permissions (should include viewer permissions + more)
		adminPermissions, err := r.AuthService.GetRolePermissions(models.RoleOrgAdmin, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.NotEmpty(t, adminPermissions)
		assert.GreaterOrEqual(t, len(adminPermissions), len(viewerPermissions))

		// Should have various actions
		actions := make(map[string]bool)
		for _, perm := range adminPermissions {
			actions[perm.Action] = true
			assert.Equal(t, models.DomainTypeOrganization, perm.DomainType)
		}
		assert.True(t, actions["read"], "Admin should have read permissions")

		// Test org owner permissions (should include admin permissions + more)
		ownerPermissions, err := r.AuthService.GetRolePermissions(models.RoleOrgOwner, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.NotEmpty(t, ownerPermissions)
		assert.GreaterOrEqual(t, len(ownerPermissions), len(adminPermissions))
	})

	t.Run("get canvas role permissions", func(t *testing.T) {
		// Test canvas viewer permissions
		viewerPermissions, err := r.AuthService.GetRolePermissions(models.RoleCanvasViewer, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.NotEmpty(t, viewerPermissions)

		// All permissions should be read-only
		for _, perm := range viewerPermissions {
			assert.Equal(t, "read", perm.Action)
			assert.Equal(t, models.DomainTypeCanvas, perm.DomainType)
		}

		// Test canvas admin permissions
		adminPermissions, err := r.AuthService.GetRolePermissions(models.RoleCanvasAdmin, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.NotEmpty(t, adminPermissions)
		assert.GreaterOrEqual(t, len(adminPermissions), len(viewerPermissions))

		// Test canvas owner permissions
		ownerPermissions, err := r.AuthService.GetRolePermissions(models.RoleCanvasOwner, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.NotEmpty(t, ownerPermissions)
		assert.GreaterOrEqual(t, len(ownerPermissions), len(adminPermissions))
	})

	t.Run("permissions include inheritance", func(t *testing.T) {
		// Canvas admin should have all viewer permissions plus admin-specific ones
		viewerPermissions, err := r.AuthService.GetRolePermissions(models.RoleCanvasViewer, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)

		adminPermissions, err := r.AuthService.GetRolePermissions(models.RoleCanvasAdmin, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)

		// Check that admin has at least all viewer permissions
		viewerPermMap := make(map[string]bool)
		for _, perm := range viewerPermissions {
			key := perm.Resource + ":" + perm.Action
			viewerPermMap[key] = true
		}

		adminPermMap := make(map[string]bool)
		for _, perm := range adminPermissions {
			key := perm.Resource + ":" + perm.Action
			adminPermMap[key] = true
		}

		// Admin should have all viewer permissions
		for viewerPerm := range viewerPermMap {
			assert.True(t, adminPermMap[viewerPerm], "Admin should have viewer permission: %s", viewerPerm)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		// Test non-existent role
		_, err := r.AuthService.GetRolePermissions("non_existent_role", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)

		// Test non-existent domain
		_, err = r.AuthService.GetRolePermissions(models.RoleOrgViewer, models.DomainTypeOrganization, "non-existent-org")
		assert.Error(t, err)

		// Test invalid domain type
		_, err = r.AuthService.GetRolePermissions(models.RoleOrgViewer, "invalid_domain", orgID)
		assert.Error(t, err)
	})
}

func Test__AuthService_GetRoleHierarchy(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()
	canvasID := r.Canvas.ID.String()

	t.Run("get organization role hierarchy", func(t *testing.T) {
		// Test org viewer hierarchy (should only include itself)
		viewerHierarchy, err := r.AuthService.GetRoleHierarchy(models.RoleOrgViewer, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Contains(t, viewerHierarchy, models.RoleOrgViewer)

		// Test org admin hierarchy (should include itself and inherited roles)
		adminHierarchy, err := r.AuthService.GetRoleHierarchy(models.RoleOrgAdmin, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Contains(t, adminHierarchy, models.RoleOrgAdmin)
		// May also include inherited roles depending on setup

		// Test org owner hierarchy (should include itself and inherited roles)
		ownerHierarchy, err := r.AuthService.GetRoleHierarchy(models.RoleOrgOwner, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Contains(t, ownerHierarchy, models.RoleOrgOwner)
		// Should be the longest hierarchy
		assert.GreaterOrEqual(t, len(ownerHierarchy), len(adminHierarchy))
	})

	t.Run("get canvas role hierarchy", func(t *testing.T) {
		// Test canvas viewer hierarchy
		viewerHierarchy, err := r.AuthService.GetRoleHierarchy(models.RoleCanvasViewer, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.Contains(t, viewerHierarchy, models.RoleCanvasViewer)

		// Test canvas admin hierarchy
		adminHierarchy, err := r.AuthService.GetRoleHierarchy(models.RoleCanvasAdmin, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.Contains(t, adminHierarchy, models.RoleCanvasAdmin)

		// Test canvas owner hierarchy
		ownerHierarchy, err := r.AuthService.GetRoleHierarchy(models.RoleCanvasOwner, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.Contains(t, ownerHierarchy, models.RoleCanvasOwner)
		// Should be the longest hierarchy
		assert.GreaterOrEqual(t, len(ownerHierarchy), len(adminHierarchy))
	})

	t.Run("hierarchy includes inheritance", func(t *testing.T) {
		// Canvas owner should include admin in hierarchy (if inheritance is set up)
		ownerHierarchy, err := r.AuthService.GetRoleHierarchy(models.RoleCanvasOwner, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)

		// The exact inheritance depends on CSV setup, but owner should have most roles
		assert.GreaterOrEqual(t, len(ownerHierarchy), 1) // At least includes itself

		// Admin should have fewer or equal roles than owner
		adminHierarchy, err := r.AuthService.GetRoleHierarchy(models.RoleCanvasAdmin, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(adminHierarchy), len(ownerHierarchy))
	})

	t.Run("hierarchy is unique", func(t *testing.T) {
		hierarchy, err := r.AuthService.GetRoleHierarchy(models.RoleOrgOwner, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)

		// Check for duplicates
		seen := make(map[string]bool)
		for _, role := range hierarchy {
			assert.False(t, seen[role], "Role %s should not appear twice in hierarchy", role)
			seen[role] = true
		}
	})

	t.Run("error cases", func(t *testing.T) {
		// Test non-existent role
		_, err := r.AuthService.GetRoleHierarchy("non_existent_role", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)

		// Test non-existent domain
		_, err = r.AuthService.GetRoleHierarchy(models.RoleOrgViewer, models.DomainTypeOrganization, "non-existent-org")
		assert.Error(t, err)

		// Test invalid domain type
		_, err = r.AuthService.GetRoleHierarchy(models.RoleOrgViewer, "invalid_domain", orgID)
		assert.Error(t, err)
	})
}

func Test__AuthService_DetectMissingPermissions(t *testing.T) {
	r := support.Setup(t)

	t.Run("detect missing permissions", func(t *testing.T) {
		newOrg, err := models.CreateOrganization("test-org", "Test Organization", "Test Organization")
		require.NoError(t, err)

		newCanvas, err := models.CreateCanvas(uuid.New(), newOrg.ID, "Test Canvas", "Test Canvas")
		require.NoError(t, err)

		missingOrgs, missingCanvases, err := r.AuthService.DetectMissingPermissions()
		require.NoError(t, err)

		// Should detect missing permissions for new out-of-sync org and canvas
		assert.Contains(t, missingOrgs, newOrg.ID.String())
		assert.Contains(t, missingCanvases, newCanvas.ID.String())
	})

	t.Run("detect no missing permissions after setup", func(t *testing.T) {
		missingOrgs, missingCanvases, err := r.AuthService.DetectMissingPermissions()
		require.NoError(t, err)
		assert.NotContains(t, missingOrgs, r.Organization.ID.String())
		assert.NotContains(t, missingCanvases, r.Canvas.ID.String())
	})
}

func Test__AuthService_SyncDefaultRoles(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()
	canvasID := r.Canvas.ID.String()

	t.Run("sync default roles for existing entities", func(t *testing.T) {
		// First check that we have missing permissions
		newOrg, err := models.CreateOrganization("test-org", "Test Organization", "Test Organization")
		require.NoError(t, err)
		_, err = models.CreateCanvas(uuid.New(), newOrg.ID, "Test Canvas", "Test Canvas")
		require.NoError(t, err)
		missingOrgsBefore, missingCanvasesBefore, err := r.AuthService.DetectMissingPermissions()
		require.NoError(t, err)

		// Sync default roles
		err = r.AuthService.SyncDefaultRoles()
		require.NoError(t, err)

		// Check that missing permissions are now resolved
		missingOrgsAfter, missingCanvasesAfter, err := r.AuthService.DetectMissingPermissions()
		require.NoError(t, err)

		// Should have fewer or same missing permissions after sync
		assert.LessOrEqual(t, len(missingOrgsAfter), len(missingOrgsBefore))
		assert.LessOrEqual(t, len(missingCanvasesAfter), len(missingCanvasesBefore))

		// Verify that roles are properly set up
		roles, err := r.AuthService.GetAllRoleDefinitions(models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3) // Should have viewer, admin, owner

		canvasRoles, err := r.AuthService.GetAllRoleDefinitions(models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(canvasRoles), 3) // Should have viewer, admin, owner
	})

	t.Run("sync is idempotent", func(t *testing.T) {
		// Run sync twice
		err := r.AuthService.SyncDefaultRoles()
		require.NoError(t, err)

		err = r.AuthService.SyncDefaultRoles()
		require.NoError(t, err)

		// Should still work and not create duplicates
		roles, err := r.AuthService.GetAllRoleDefinitions(models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3)

		// Test that permissions still work
		userID := r.User.String()
		err = r.AuthService.AssignRole(userID, models.RoleOrgViewer, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		allowed, err := r.AuthService.CheckOrganizationPermission(userID, orgID, "canvas", "read")
		require.NoError(t, err)
		assert.True(t, allowed)
	})
}

func Test__AuthService_CheckAndSyncMissingPermissions(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	org, err := models.CreateOrganization("test-org", "Test Organization", "Test Organization")
	require.NoError(t, err)
	canvas, err := models.CreateCanvas(uuid.New(), org.ID, "Test Canvas", "Test Canvas")
	require.NoError(t, err)

	t.Run("check and sync in one operation", func(t *testing.T) {
		// Run the combined operation
		err := authService.CheckAndSyncMissingPermissions()
		require.NoError(t, err)

		// Test org permissions
		roles, err := authService.GetAllRoleDefinitions(models.DomainTypeOrganization, org.ID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3)

		// Test canvas permissions
		canvasRoles, err := authService.GetAllRoleDefinitions(models.DomainTypeCanvas, canvas.ID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(canvasRoles), 3)

		// Test that roles work properly
		userID := uuid.New().String()
		err = authService.AssignRole(userID, models.RoleOrgAdmin, org.ID.String(), models.DomainTypeOrganization)
		require.NoError(t, err)

		allowed, err := authService.CheckOrganizationPermission(userID, org.ID.String(), "canvas", "create")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("no errors on already synced system", func(t *testing.T) {
		// Run sync twice - should not error
		err := authService.CheckAndSyncMissingPermissions()
		require.NoError(t, err)

		err = authService.CheckAndSyncMissingPermissions()
		require.NoError(t, err)
	})
}

func Test__AuthService_SyncOrganizationRoles(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := uuid.New()
	org, err := models.CreateOrganization("test-org", "Test Organization", "Test Organization")
	require.NoError(t, err)
	orgID := org.ID.String()

	t.Run("sync organization roles creates expected policies", func(t *testing.T) {
		// Sync org roles
		err := authService.SyncOrganizationRoles(orgID)
		require.NoError(t, err)

		// Test that all expected roles exist
		expectedRoles := []string{models.RoleOrgViewer, models.RoleOrgAdmin, models.RoleOrgOwner}
		for _, role := range expectedRoles {
			roleDef, err := authService.GetRoleDefinition(role, models.DomainTypeOrganization, orgID)
			require.NoError(t, err)
			assert.Equal(t, role, roleDef.Name)
			assert.NotEmpty(t, roleDef.Permissions)
		}

		// Test role hierarchy
		err = authService.AssignRole(userID.String(), models.RoleOrgOwner, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Owner should have admin and viewer permissions through inheritance
		roles, err := authService.GetUserRolesForOrg(userID.String(), orgID)
		require.NoError(t, err)

		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		assert.True(t, flatRoles[models.RoleOrgOwner])
		assert.True(t, flatRoles[models.RoleOrgAdmin])
		assert.True(t, flatRoles[models.RoleOrgViewer])
	})

	t.Run("sync is idempotent for organizations", func(t *testing.T) {
		// Sync multiple times
		err := authService.SyncOrganizationRoles(orgID)
		require.NoError(t, err)

		err = authService.SyncOrganizationRoles(orgID)
		require.NoError(t, err)

		err = authService.SyncOrganizationRoles(orgID)
		require.NoError(t, err)

		// Should still work correctly
		roles, err := authService.GetAllRoleDefinitions(models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3)
	})
}

func Test__AuthService_SyncCanvasRoles(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := uuid.New()
	org, err := models.CreateOrganization("test-org", "Test Organization", "Test Organization")
	require.NoError(t, err)
	canvas, err := models.CreateCanvas(userID, org.ID, "Test Canvas", "Test Canvas")
	require.NoError(t, err)

	t.Run("sync canvas roles creates expected policies", func(t *testing.T) {
		err := authService.SyncCanvasRoles(canvas.ID.String())
		require.NoError(t, err)

		// Test that all expected roles exist
		expectedRoles := []string{models.RoleCanvasViewer, models.RoleCanvasAdmin, models.RoleCanvasOwner}
		for _, role := range expectedRoles {
			roleDef, err := authService.GetRoleDefinition(role, models.DomainTypeCanvas, canvas.ID.String())
			require.NoError(t, err)
			assert.Equal(t, role, roleDef.Name)
			assert.NotEmpty(t, roleDef.Permissions)
		}

		// Test role hierarchy
		err = authService.AssignRole(userID.String(), models.RoleCanvasOwner, canvas.ID.String(), models.DomainTypeCanvas)
		require.NoError(t, err)

		// Owner should have admin and viewer permissions through inheritance
		roles, err := authService.GetUserRolesForCanvas(userID.String(), canvas.ID.String())
		require.NoError(t, err)

		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		assert.True(t, flatRoles[models.RoleCanvasOwner])
		assert.True(t, flatRoles[models.RoleCanvasAdmin])
		assert.True(t, flatRoles[models.RoleCanvasViewer])
	})

	t.Run("sync is idempotent for canvases", func(t *testing.T) {
		// Sync multiple times
		err := authService.SyncCanvasRoles(canvas.ID.String())
		require.NoError(t, err)

		err = authService.SyncCanvasRoles(canvas.ID.String())
		require.NoError(t, err)

		err = authService.SyncCanvasRoles(canvas.ID.String())
		require.NoError(t, err)

		// Should still work correctly
		roles, err := authService.GetAllRoleDefinitions(models.DomainTypeCanvas, canvas.ID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3)
	})
}
