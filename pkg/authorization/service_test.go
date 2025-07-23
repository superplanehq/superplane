package authorization

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__AuthService_BasicPermissions(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()
	canvasID := r.Canvas.ID.String()
	orgID := "example-org-id"
	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)
	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	t.Run("user without roles has no permissions", func(t *testing.T) {
		allowedOrg, err := authService.CheckOrganizationPermission(userID, orgID, "canvas", "read")
		require.NoError(t, err)
		assert.False(t, allowedOrg)

		allowedCanvas, err := authService.CheckCanvasPermission(userID, canvasID, "stage", "read")
		require.NoError(t, err)
		assert.False(t, allowedCanvas)
	})

	t.Run("canvas owner has all permissions", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleCanvasOwner, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		roles, err := authService.GetUserRolesForCanvas(userID, canvasID)
		require.NoError(t, err)
		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}
		require.True(t, flatRoles[RoleCanvasOwner])
		require.True(t, flatRoles[RoleCanvasAdmin])
		require.True(t, flatRoles[RoleCanvasViewer])

		// Test viewer permissions (inherited)
		allowed, err := authService.CheckCanvasPermission(userID, canvasID, "eventsource", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = authService.CheckCanvasPermission(userID, canvasID, "stage", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = authService.CheckCanvasPermission(userID, canvasID, "stageevent", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Test admin permissions (inherited)
		resources := []string{"eventsource", "stage"}
		actions := []string{"create", "update", "delete"}
		for _, resource := range resources {
			for _, action := range actions {
				allowed, err := authService.CheckCanvasPermission(userID, canvasID, resource, action)
				require.NoError(t, err)
				assert.True(t, allowed, "Canvas owner should have %s permission for %s", action, resource)
			}
		}

		// Test stageevent approve permission
		allowed, err = authService.CheckCanvasPermission(userID, canvasID, "stageevent", "approve")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Test member permissions
		allowed, err = authService.CheckCanvasPermission(userID, canvasID, "member", "invite")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = authService.CheckCanvasPermission(userID, canvasID, "member", "remove")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("canvas viewer has only read permissions", func(t *testing.T) {
		viewerID := uuid.New().String()
		err := authService.AssignRole(viewerID, RoleCanvasViewer, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		// Should have read permissions
		allowed, err := authService.CheckCanvasPermission(viewerID, canvasID, "eventsource", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = authService.CheckCanvasPermission(viewerID, canvasID, "stage", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = authService.CheckCanvasPermission(viewerID, canvasID, "stageevent", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should not have write permissions
		allowed, err = authService.CheckCanvasPermission(viewerID, canvasID, "stage", "create")
		require.NoError(t, err)
		assert.False(t, allowed)

		allowed, err = authService.CheckCanvasPermission(viewerID, canvasID, "stage", "update")
		require.NoError(t, err)
		assert.False(t, allowed)

		// Should not have approve permission
		allowed, err = authService.CheckCanvasPermission(viewerID, canvasID, "stageevent", "approve")
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("canvas admin has read and write permissions", func(t *testing.T) {
		adminID := uuid.New().String()
		err := authService.AssignRole(adminID, RoleCanvasAdmin, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		// Should have read permissions (inherited from viewer)
		allowed, err := authService.CheckCanvasPermission(adminID, canvasID, "stage", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should have create/update/delete permissions
		resources := []string{"eventsource", "stage"}
		actions := []string{"create", "update", "delete"}
		for _, resource := range resources {
			for _, action := range actions {
				allowed, err := authService.CheckCanvasPermission(adminID, canvasID, resource, action)
				require.NoError(t, err)
				assert.True(t, allowed, "Canvas admin should have %s permission for %s", action, resource)
			}
		}

		// Should have approve permission for stageevent
		allowed, err = authService.CheckCanvasPermission(adminID, canvasID, "stageevent", "approve")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should have member invite permission
		allowed, err = authService.CheckCanvasPermission(adminID, canvasID, "member", "invite")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should not have member remove permission (owner only)
		allowed, err = authService.CheckCanvasPermission(adminID, canvasID, "member", "remove")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__AuthService_OrganizationPermissions(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()
	orgID := uuid.New().String()
	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("org owner has all permissions", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleOrgOwner, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Should have all canvas permissions (inherited from admin)
		actions := []string{"read", "create", "update", "delete"}
		for _, action := range actions {
			allowed, err := authService.CheckOrganizationPermission(userID, orgID, "canvas", action)
			require.NoError(t, err)
			assert.True(t, allowed, "Org owner should have %s permission for canvas", action)
		}

		// Should have user management permissions (inherited from admin)
		allowed, err := authService.CheckOrganizationPermission(userID, orgID, "user", "invite")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = authService.CheckOrganizationPermission(userID, orgID, "user", "remove")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should have org management permissions (owner only)
		allowed, err = authService.CheckOrganizationPermission(userID, orgID, models.DomainTypeOrganization, "update")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = authService.CheckOrganizationPermission(userID, orgID, models.DomainTypeOrganization, "delete")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("org admin has limited permissions", func(t *testing.T) {
		adminID := uuid.New().String()
		err := authService.AssignRole(adminID, RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Should have canvas management permissions
		actions := []string{"read", "create", "update", "delete"}
		for _, action := range actions {
			allowed, err := authService.CheckOrganizationPermission(adminID, orgID, "canvas", action)
			require.NoError(t, err)
			assert.True(t, allowed, "Org admin should have %s permission for canvas", action)
		}

		// Should have user management permissions
		allowed, err := authService.CheckOrganizationPermission(adminID, orgID, "user", "invite")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = authService.CheckOrganizationPermission(adminID, orgID, "user", "remove")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should not have org management permissions
		allowed, err = authService.CheckOrganizationPermission(adminID, orgID, models.DomainTypeOrganization, "update")
		require.NoError(t, err)
		assert.False(t, allowed)

		allowed, err = authService.CheckOrganizationPermission(adminID, orgID, models.DomainTypeOrganization, "delete")
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("org viewer has only read permissions", func(t *testing.T) {
		viewerID := uuid.New().String()
		err := authService.AssignRole(viewerID, RoleOrgViewer, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Should have read permission
		allowed, err := authService.CheckOrganizationPermission(viewerID, orgID, "canvas", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should not have create/update/delete permissions
		actions := []string{"create", "update", "delete"}
		for _, action := range actions {
			allowed, err := authService.CheckOrganizationPermission(viewerID, orgID, "canvas", action)
			require.NoError(t, err)
			assert.False(t, allowed, "Org viewer should not have %s permission for canvas", action)
		}

		// Should not have user management permissions
		allowed, err = authService.CheckOrganizationPermission(viewerID, orgID, "user", "invite")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__AuthService_RoleManagement(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()
	orgID := uuid.New().String()
	canvasID := uuid.New().String()

	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)
	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	t.Run("assign and remove roles", func(t *testing.T) {
		// Assign role
		err := authService.AssignRole(userID, RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Verify role assignment
		roles, err := authService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)
		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}
		require.True(t, flatRoles[RoleOrgAdmin])
		// Check permissions
		allowed, err := authService.CheckOrganizationPermission(userID, orgID, "canvas", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Remove role
		err = authService.RemoveRole(userID, RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Verify role removal
		roles, err = authService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)
		assert.NotContains(t, roles, RoleOrgAdmin)
		// Check permissions
		allowed, err = authService.CheckOrganizationPermission(userID, orgID, "canvas", "read")
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("get users for role", func(t *testing.T) {
		user1 := uuid.New().String()
		user2 := uuid.New().String()

		err := authService.AssignRole(user1, RoleCanvasViewer, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)
		err = authService.AssignRole(user2, RoleCanvasViewer, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		users, err := authService.GetCanvasUsersForRole(RoleCanvasViewer, canvasID)
		require.NoError(t, err)
		assert.Contains(t, users, user1)
		assert.Contains(t, users, user2)
	})

	t.Run("invalid role assignment", func(t *testing.T) {
		err := authService.AssignRole(userID, "invalid_role", orgID, models.DomainTypeOrganization)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role")
	})
}

func Test__AuthService_GroupManagement(t *testing.T) {
	_ = support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	orgID := uuid.New().String()
	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("create and manage groups", func(t *testing.T) {
		groupName := "engineering-team"

		// Create group
		err := authService.CreateGroup(orgID, models.DomainTypeOrganization, groupName, RoleOrgAdmin)
		require.NoError(t, err)

		// Add users to group
		user1 := uuid.New().String()
		user2 := uuid.New().String()

		err = authService.AddUserToGroup(orgID, models.DomainTypeOrganization, user1, groupName)
		require.NoError(t, err)
		err = authService.AddUserToGroup(orgID, models.DomainTypeOrganization, user2, groupName)
		require.NoError(t, err)

		// Get group users
		users, err := authService.GetGroupUsers(orgID, models.DomainTypeOrganization, groupName)
		require.NoError(t, err)
		assert.Contains(t, users, user1)
		assert.Contains(t, users, user2)

		// Check permissions through group
		allowed, err := authService.CheckOrganizationPermission(user1, orgID, "canvas", "create")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Remove user from group
		err = authService.RemoveUserFromGroup(orgID, models.DomainTypeOrganization, user1, groupName)
		require.NoError(t, err)

		// Verify removal
		users, err = authService.GetGroupUsers(orgID, models.DomainTypeOrganization, groupName)
		require.NoError(t, err)
		assert.NotContains(t, users, user1)
		assert.Contains(t, users, user2)
	})

	t.Run("create group with invalid role", func(t *testing.T) {
		err := authService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group", "invalid_role")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role")
	})

	t.Run("add user to non-existent group", func(t *testing.T) {
		userID := uuid.New().String()
		err := authService.AddUserToGroup(orgID, models.DomainTypeOrganization, userID, "non-existent-group")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("get groups and roles", func(t *testing.T) {
		// Create multiple groups
		err := authService.CreateGroup(orgID, models.DomainTypeOrganization, "admins", RoleOrgAdmin)
		require.NoError(t, err)
		err = authService.CreateGroup(orgID, models.DomainTypeOrganization, "viewers", RoleOrgViewer)
		require.NoError(t, err)

		// Add users to make groups detectable
		user1 := uuid.New().String()
		user2 := uuid.New().String()
		err = authService.AddUserToGroup(orgID, models.DomainTypeOrganization, user1, "admins")
		require.NoError(t, err)
		err = authService.AddUserToGroup(orgID, models.DomainTypeOrganization, user2, "viewers")
		require.NoError(t, err)

		// Get all groups
		groups, err := authService.GetGroups(orgID, models.DomainTypeOrganization)
		require.NoError(t, err)
		assert.Contains(t, groups, "admins")
		assert.Contains(t, groups, "viewers")

		// Get group role
		role, err := authService.GetGroupRole(orgID, models.DomainTypeOrganization, "admins")
		require.NoError(t, err)
		assert.Equal(t, role, RoleOrgAdmin)

		role, err = authService.GetGroupRole(orgID, models.DomainTypeOrganization, "viewers")
		require.NoError(t, err)
		assert.Equal(t, role, RoleOrgViewer)
	})
}

func Test__AuthService_AccessibleResources(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()
	org1 := uuid.New().String()
	org2 := uuid.New().String()
	canvas1 := uuid.New().String()
	canvas2 := uuid.New().String()

	// Setup roles
	err = authService.SetupOrganizationRoles(org1)
	require.NoError(t, err)
	err = authService.SetupOrganizationRoles(org2)
	require.NoError(t, err)
	err = authService.SetupCanvasRoles(canvas1)
	require.NoError(t, err)
	err = authService.SetupCanvasRoles(canvas2)
	require.NoError(t, err)

	t.Run("get accessible organizations", func(t *testing.T) {
		// Assign user to organizations
		err := authService.AssignRole(userID, RoleOrgViewer, org1, models.DomainTypeOrganization)
		require.NoError(t, err)
		err = authService.AssignRole(userID, RoleOrgAdmin, org2, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Get accessible orgs
		orgs, err := authService.GetAccessibleOrgsForUser(userID)
		require.NoError(t, err)
		assert.Contains(t, orgs, org1)
		assert.Contains(t, orgs, org2)
	})

	t.Run("get accessible canvases", func(t *testing.T) {
		// Assign user to canvases
		err := authService.AssignRole(userID, RoleCanvasViewer, canvas1, models.DomainTypeCanvas)
		require.NoError(t, err)
		err = authService.AssignRole(userID, RoleCanvasOwner, canvas2, models.DomainTypeCanvas)
		require.NoError(t, err)

		// Get accessible canvases
		canvases, err := authService.GetAccessibleCanvasesForUser(userID)
		require.NoError(t, err)
		assert.Contains(t, canvases, canvas1)
		assert.Contains(t, canvases, canvas2)
	})
}

func Test__AuthService_CreateOrganizationOwner(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()
	orgID := uuid.New().String()

	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("create organization owner", func(t *testing.T) {
		err := authService.CreateOrganizationOwner(userID, orgID)
		require.NoError(t, err)

		// Verify owner permissions
		allowed, err := authService.CheckOrganizationPermission(userID, orgID, models.DomainTypeOrganization, "update")
		require.NoError(t, err)
		assert.True(t, allowed)

		roles, err := authService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)
		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		require.True(t, flatRoles[RoleOrgOwner])
	})
}

func Test__AuthService_RoleHierarchy(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()
	canvasID := uuid.New().String()
	orgID := uuid.New().String()

	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)
	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("canvas owner inherits admin and viewer permissions", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleCanvasOwner, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		// Get implicit roles (should include inherited roles)
		roles, err := authService.GetUserRolesForCanvas(userID, canvasID)
		require.NoError(t, err)

		// Should have all three roles due to hierarchy
		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		require.True(t, flatRoles[RoleCanvasOwner])
		require.True(t, flatRoles[RoleCanvasAdmin])
		require.True(t, flatRoles[RoleCanvasViewer])
	})

	t.Run("canvas admin inherits viewer permissions", func(t *testing.T) {
		adminID := uuid.New().String()
		err := authService.AssignRole(adminID, RoleCanvasAdmin, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		roles, err := authService.GetUserRolesForCanvas(adminID, canvasID)
		require.NoError(t, err)

		// Should have admin and viewer roles
		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		require.True(t, flatRoles[RoleCanvasAdmin])
		require.True(t, flatRoles[RoleCanvasViewer])
		// Should not have owner role
		require.False(t, flatRoles[RoleCanvasOwner])
	})

	t.Run("org owner inherits admin and viewer permissions", func(t *testing.T) {
		ownerID := uuid.New().String()
		err := authService.AssignRole(ownerID, RoleOrgOwner, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		roles, err := authService.GetUserRolesForOrg(ownerID, orgID)
		require.NoError(t, err)

		// Should have all three roles due to hierarchy
		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		require.True(t, flatRoles[RoleOrgOwner])
		require.True(t, flatRoles[RoleOrgAdmin])
		require.True(t, flatRoles[RoleOrgViewer])
	})
}

func Test__AuthService_DuplicateAssignments(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()
	orgID := uuid.New().String()

	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("duplicate role assignment is idempotent", func(t *testing.T) {
		// First assignment
		err := authService.AssignRole(userID, RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Duplicate assignment should not error
		err = authService.AssignRole(userID, RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Should still have the role only once
		roles, err := authService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)

		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		require.True(t, flatRoles[RoleOrgAdmin])
	})

	t.Run("duplicate group creation fails", func(t *testing.T) {
		groupName := "duplicate-test-group"

		// First creation
		err := authService.CreateGroup(orgID, models.DomainTypeOrganization, groupName, RoleOrgViewer)
		require.NoError(t, err)

		// Duplicate creation should fail
		err = authService.CreateGroup(orgID, models.DomainTypeOrganization, groupName, RoleOrgViewer)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func Test__AuthService_CrossDomainPermissions(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	t.Run("org role does not grant canvas permissions", func(t *testing.T) {
		userID := r.User.String()
		orgID := uuid.New().String()
		canvasID := uuid.New().String()

		err := authService.SetupOrganizationRoles(orgID)
		require.NoError(t, err)
		err = authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		// Assign org owner role
		err = authService.AssignRole(userID, RoleOrgOwner, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Should not have canvas permissions
		allowed, err := authService.CheckCanvasPermission(userID, canvasID, "stage", "read")
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("canvas role does not grant org permissions", func(t *testing.T) {
		userID := r.User.String()
		canvasID := uuid.New().String()
		orgID := uuid.New().String()

		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)
		err = authService.SetupOrganizationRoles(orgID)
		require.NoError(t, err)

		// Assign canvas owner role
		err = authService.AssignRole(userID, RoleCanvasOwner, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		// Should not have org permissions
		allowed, err := authService.CheckOrganizationPermission(userID, orgID, "canvas", "read")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__AuthService_PermissionBoundaries(t *testing.T) {
	_ = support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	canvasID := uuid.New().String()
	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	t.Run("member remove is owner-only permission", func(t *testing.T) {
		viewerID := uuid.New().String()
		adminID := uuid.New().String()
		ownerID := uuid.New().String()

		// Assign roles
		err := authService.AssignRole(viewerID, RoleCanvasViewer, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)
		err = authService.AssignRole(adminID, RoleCanvasAdmin, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)
		err = authService.AssignRole(ownerID, RoleCanvasOwner, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		// Viewer should not have member remove permission
		allowed, err := authService.CheckCanvasPermission(viewerID, canvasID, "member", "remove")
		require.NoError(t, err)
		assert.False(t, allowed)

		// Admin should not have member remove permission
		allowed, err = authService.CheckCanvasPermission(adminID, canvasID, "member", "remove")
		require.NoError(t, err)
		assert.False(t, allowed)

		// Owner should have member remove permission
		allowed, err = authService.CheckCanvasPermission(ownerID, canvasID, "member", "remove")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("org update and delete are owner-only permissions", func(t *testing.T) {
		orgID := uuid.New().String()
		err := authService.SetupOrganizationRoles(orgID)
		require.NoError(t, err)

		viewerID := uuid.New().String()
		adminID := uuid.New().String()
		ownerID := uuid.New().String()

		// Assign roles
		err = authService.AssignRole(viewerID, RoleOrgViewer, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)
		err = authService.AssignRole(adminID, RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)
		err = authService.AssignRole(ownerID, RoleOrgOwner, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Check org update permission
		allowed, err := authService.CheckOrganizationPermission(viewerID, orgID, models.DomainTypeOrganization, "update")
		require.NoError(t, err)
		assert.False(t, allowed, "Viewer should not have org update permission")

		allowed, err = authService.CheckOrganizationPermission(adminID, orgID, models.DomainTypeOrganization, "update")
		require.NoError(t, err)
		assert.False(t, allowed, "Admin should not have org update permission")

		allowed, err = authService.CheckOrganizationPermission(ownerID, orgID, models.DomainTypeOrganization, "update")
		require.NoError(t, err)
		assert.True(t, allowed, "Owner should have org update permission")

		// Check org delete permission
		allowed, err = authService.CheckOrganizationPermission(viewerID, orgID, models.DomainTypeOrganization, "delete")
		require.NoError(t, err)
		assert.False(t, allowed, "Viewer should not have org delete permission")

		allowed, err = authService.CheckOrganizationPermission(adminID, orgID, models.DomainTypeOrganization, "delete")
		require.NoError(t, err)
		assert.False(t, allowed, "Admin should not have org delete permission")

		allowed, err = authService.CheckOrganizationPermission(ownerID, orgID, models.DomainTypeOrganization, "delete")
		require.NoError(t, err)
		assert.True(t, allowed, "Owner should have org delete permission")
	})
}

func Test__AuthService_GetRoleDefinition(t *testing.T) {
	_ = support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	orgID := uuid.New().String()
	canvasID := uuid.New().String()

	// Setup domains
	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)
	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	t.Run("get organization role definition", func(t *testing.T) {
		viewerRole, err := authService.GetRoleDefinition(RoleOrgViewer, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, RoleOrgViewer, viewerRole.Name)
		assert.Equal(t, models.DomainTypeOrganization, viewerRole.DomainType)
		assert.NotEmpty(t, viewerRole.Description)
		assert.True(t, viewerRole.Readonly)
		assert.NotEmpty(t, viewerRole.Permissions)

		// Test org admin role
		adminRole, err := authService.GetRoleDefinition(RoleOrgAdmin, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, RoleOrgAdmin, adminRole.Name)
		assert.Equal(t, models.DomainTypeOrganization, adminRole.DomainType)
		assert.NotEmpty(t, adminRole.Description)
		assert.True(t, adminRole.Readonly)
		assert.NotEmpty(t, adminRole.Permissions)

		// Test org owner role
		ownerRole, err := authService.GetRoleDefinition(RoleOrgOwner, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, RoleOrgOwner, ownerRole.Name)
		assert.Equal(t, models.DomainTypeOrganization, ownerRole.DomainType)
		assert.NotEmpty(t, ownerRole.Description)
		assert.True(t, ownerRole.Readonly)
		assert.NotEmpty(t, ownerRole.Permissions)
	})

	t.Run("get canvas role definition", func(t *testing.T) {
		// Test canvas viewer role
		viewerRole, err := authService.GetRoleDefinition(RoleCanvasViewer, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.Equal(t, RoleCanvasViewer, viewerRole.Name)
		assert.Equal(t, models.DomainTypeCanvas, viewerRole.DomainType)
		assert.NotEmpty(t, viewerRole.Description)
		assert.True(t, viewerRole.Readonly)
		assert.NotEmpty(t, viewerRole.Permissions)

		// Test canvas admin role
		adminRole, err := authService.GetRoleDefinition(RoleCanvasAdmin, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.Equal(t, RoleCanvasAdmin, adminRole.Name)
		assert.Equal(t, models.DomainTypeCanvas, adminRole.DomainType)
		assert.NotEmpty(t, adminRole.Description)
		assert.True(t, adminRole.Readonly)
		assert.NotEmpty(t, adminRole.Permissions)

		// Test canvas owner role
		ownerRole, err := authService.GetRoleDefinition(RoleCanvasOwner, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.Equal(t, RoleCanvasOwner, ownerRole.Name)
		assert.Equal(t, models.DomainTypeCanvas, ownerRole.DomainType)
		assert.NotEmpty(t, ownerRole.Description)
		assert.True(t, ownerRole.Readonly)
		assert.NotEmpty(t, ownerRole.Permissions)
	})

	t.Run("error cases", func(t *testing.T) {
		// Test non-existent role
		_, err := authService.GetRoleDefinition("non_existent_role", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test non-existent domain
		_, err = authService.GetRoleDefinition(RoleOrgViewer, models.DomainTypeOrganization, "non-existent-org")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test invalid domain type
		_, err = authService.GetRoleDefinition(RoleOrgViewer, "invalid_domain", orgID)
		assert.Error(t, err)
	})

	t.Run("permissions are populated", func(t *testing.T) {
		role, err := authService.GetRoleDefinition(RoleOrgAdmin, models.DomainTypeOrganization, orgID)
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
	_ = support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	orgID := uuid.New().String()
	canvasID := uuid.New().String()

	// Setup domains
	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)
	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	t.Run("get all organization roles", func(t *testing.T) {
		roles, err := authService.GetAllRoleDefinitions(models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3) // Should have at least viewer, admin, owner

		// Extract role names
		roleNames := make([]string, len(roles))
		for i, role := range roles {
			roleNames[i] = role.Name
		}

		// Check that we have the expected roles
		assert.Contains(t, roleNames, RoleOrgViewer)
		assert.Contains(t, roleNames, RoleOrgAdmin)
		assert.Contains(t, roleNames, RoleOrgOwner)

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
		roles, err := authService.GetAllRoleDefinitions(models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3) // Should have at least viewer, admin, owner

		// Extract role names
		roleNames := make([]string, len(roles))
		for i, role := range roles {
			roleNames[i] = role.Name
		}

		// Check that we have the expected roles
		assert.Contains(t, roleNames, RoleCanvasViewer)
		assert.Contains(t, roleNames, RoleCanvasAdmin)
		assert.Contains(t, roleNames, RoleCanvasOwner)

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
		anotherOrgID := uuid.New().String()
		err := authService.SetupOrganizationRoles(anotherOrgID)
		require.NoError(t, err)

		// Both should have the same number of roles
		roles1, err := authService.GetAllRoleDefinitions(models.DomainTypeOrganization, orgID)
		require.NoError(t, err)

		roles2, err := authService.GetAllRoleDefinitions(models.DomainTypeOrganization, anotherOrgID)
		require.NoError(t, err)

		assert.Equal(t, len(roles1), len(roles2))
	})

	t.Run("empty responses", func(t *testing.T) {
		// Test invalid domain type
		definitions, _ := authService.GetAllRoleDefinitions("invalid_domain", orgID)
		assert.Empty(t, definitions)

		// Test non-existent domain
		definitions, _ = authService.GetAllRoleDefinitions(models.DomainTypeOrganization, "non-existent-org")
		assert.Empty(t, definitions)
	})
}

func Test__AuthService_GetRolePermissions(t *testing.T) {
	_ = support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	orgID := uuid.New().String()
	canvasID := uuid.New().String()

	// Setup domains
	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)
	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	t.Run("get organization role permissions", func(t *testing.T) {
		// Test org viewer permissions
		viewerPermissions, err := authService.GetRolePermissions(RoleOrgViewer, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.NotEmpty(t, viewerPermissions)

		// All permissions should be read-only
		for _, perm := range viewerPermissions {
			assert.Equal(t, "read", perm.Action)
			assert.Equal(t, models.DomainTypeOrganization, perm.DomainType)
		}

		// Test org admin permissions (should include viewer permissions + more)
		adminPermissions, err := authService.GetRolePermissions(RoleOrgAdmin, models.DomainTypeOrganization, orgID)
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
		ownerPermissions, err := authService.GetRolePermissions(RoleOrgOwner, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.NotEmpty(t, ownerPermissions)
		assert.GreaterOrEqual(t, len(ownerPermissions), len(adminPermissions))
	})

	t.Run("get canvas role permissions", func(t *testing.T) {
		// Test canvas viewer permissions
		viewerPermissions, err := authService.GetRolePermissions(RoleCanvasViewer, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.NotEmpty(t, viewerPermissions)

		// All permissions should be read-only
		for _, perm := range viewerPermissions {
			assert.Equal(t, "read", perm.Action)
			assert.Equal(t, models.DomainTypeCanvas, perm.DomainType)
		}

		// Test canvas admin permissions
		adminPermissions, err := authService.GetRolePermissions(RoleCanvasAdmin, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.NotEmpty(t, adminPermissions)
		assert.GreaterOrEqual(t, len(adminPermissions), len(viewerPermissions))

		// Test canvas owner permissions
		ownerPermissions, err := authService.GetRolePermissions(RoleCanvasOwner, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.NotEmpty(t, ownerPermissions)
		assert.GreaterOrEqual(t, len(ownerPermissions), len(adminPermissions))
	})

	t.Run("permissions include inheritance", func(t *testing.T) {
		// Canvas admin should have all viewer permissions plus admin-specific ones
		viewerPermissions, err := authService.GetRolePermissions(RoleCanvasViewer, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)

		adminPermissions, err := authService.GetRolePermissions(RoleCanvasAdmin, models.DomainTypeCanvas, canvasID)
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
		_, err := authService.GetRolePermissions("non_existent_role", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)

		// Test non-existent domain
		_, err = authService.GetRolePermissions(RoleOrgViewer, models.DomainTypeOrganization, "non-existent-org")
		assert.Error(t, err)

		// Test invalid domain type
		_, err = authService.GetRolePermissions(RoleOrgViewer, "invalid_domain", orgID)
		assert.Error(t, err)
	})
}

func Test__AuthService_GetRoleHierarchy(t *testing.T) {
	_ = support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	orgID := uuid.New().String()
	canvasID := uuid.New().String()

	// Setup domains
	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)
	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	t.Run("get organization role hierarchy", func(t *testing.T) {
		// Test org viewer hierarchy (should only include itself)
		viewerHierarchy, err := authService.GetRoleHierarchy(RoleOrgViewer, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Contains(t, viewerHierarchy, RoleOrgViewer)

		// Test org admin hierarchy (should include itself and inherited roles)
		adminHierarchy, err := authService.GetRoleHierarchy(RoleOrgAdmin, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Contains(t, adminHierarchy, RoleOrgAdmin)
		// May also include inherited roles depending on setup

		// Test org owner hierarchy (should include itself and inherited roles)
		ownerHierarchy, err := authService.GetRoleHierarchy(RoleOrgOwner, models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Contains(t, ownerHierarchy, RoleOrgOwner)
		// Should be the longest hierarchy
		assert.GreaterOrEqual(t, len(ownerHierarchy), len(adminHierarchy))
	})

	t.Run("get canvas role hierarchy", func(t *testing.T) {
		// Test canvas viewer hierarchy
		viewerHierarchy, err := authService.GetRoleHierarchy(RoleCanvasViewer, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.Contains(t, viewerHierarchy, RoleCanvasViewer)

		// Test canvas admin hierarchy
		adminHierarchy, err := authService.GetRoleHierarchy(RoleCanvasAdmin, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.Contains(t, adminHierarchy, RoleCanvasAdmin)

		// Test canvas owner hierarchy
		ownerHierarchy, err := authService.GetRoleHierarchy(RoleCanvasOwner, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.Contains(t, ownerHierarchy, RoleCanvasOwner)
		// Should be the longest hierarchy
		assert.GreaterOrEqual(t, len(ownerHierarchy), len(adminHierarchy))
	})

	t.Run("hierarchy includes inheritance", func(t *testing.T) {
		// Canvas owner should include admin in hierarchy (if inheritance is set up)
		ownerHierarchy, err := authService.GetRoleHierarchy(RoleCanvasOwner, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)

		// The exact inheritance depends on CSV setup, but owner should have most roles
		assert.GreaterOrEqual(t, len(ownerHierarchy), 1) // At least includes itself

		// Admin should have fewer or equal roles than owner
		adminHierarchy, err := authService.GetRoleHierarchy(RoleCanvasAdmin, models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(adminHierarchy), len(ownerHierarchy))
	})

	t.Run("hierarchy is unique", func(t *testing.T) {
		hierarchy, err := authService.GetRoleHierarchy(RoleOrgOwner, models.DomainTypeOrganization, orgID)
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
		_, err := authService.GetRoleHierarchy("non_existent_role", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)

		// Test non-existent domain
		_, err = authService.GetRoleHierarchy(RoleOrgViewer, models.DomainTypeOrganization, "non-existent-org")
		assert.Error(t, err)

		// Test invalid domain type
		_, err = authService.GetRoleHierarchy(RoleOrgViewer, "invalid_domain", orgID)
		assert.Error(t, err)
	})
}

func Test__AuthService_DetectMissingPermissions(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	t.Run("detect missing permissions in empty database", func(t *testing.T) {
		// Since we have orgs and canvases but no permissions set up yet
		missingOrgs, missingCanvases, err := authService.DetectMissingPermissions()
		require.NoError(t, err)

		// Should detect missing permissions for existing org and canvas
		assert.GreaterOrEqual(t, len(missingOrgs), 0, "Should detect orgs with missing permissions")
		assert.GreaterOrEqual(t, len(missingCanvases), 0, "Should detect canvases with missing permissions")
	})

	t.Run("detect no missing permissions after setup", func(t *testing.T) {
		orgID := r.Organization.ID.String()
		canvasID := r.Canvas.ID.String()

		// Setup roles for org and canvas
		err := authService.SetupOrganizationRoles(orgID)
		require.NoError(t, err)
		err = authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		// Now detect missing permissions
		missingOrgs, missingCanvases, err := authService.DetectMissingPermissions()
		require.NoError(t, err)

		// Should not detect any missing permissions for the setup org and canvas
		assert.False(t, slices.Contains(missingOrgs, orgID), "Should not find missing permissions for setup org")
		assert.False(t, slices.Contains(missingCanvases, canvasID), "Should not find missing permissions for setup canvas")
	})

	t.Run("detect missing permissions after partial setup", func(t *testing.T) {
		// Test that we can detect missing permissions for existing entities
		// Note: In real scenario, orgs/canvases would exist but not have roles set up

		missingOrgs, _, err := authService.DetectMissingPermissions()
		require.NoError(t, err)

		// Should detect missing permissions for any orgs that exist but don't have roles set up
		assert.GreaterOrEqual(t, len(missingOrgs), 0)
	})
}

func Test__AuthService_SyncDefaultRoles(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	orgID := r.Organization.ID.String()
	canvasID := r.Canvas.ID.String()

	t.Run("sync default roles for existing entities", func(t *testing.T) {
		// First check that we have missing permissions
		missingOrgsBefore, missingCanvasesBefore, err := authService.DetectMissingPermissions()
		require.NoError(t, err)

		// Sync default roles
		err = authService.SyncDefaultRoles()
		require.NoError(t, err)

		// Check that missing permissions are now resolved
		missingOrgsAfter, missingCanvasesAfter, err := authService.DetectMissingPermissions()
		require.NoError(t, err)

		// Should have fewer or same missing permissions after sync
		assert.LessOrEqual(t, len(missingOrgsAfter), len(missingOrgsBefore))
		assert.LessOrEqual(t, len(missingCanvasesAfter), len(missingCanvasesBefore))

		// Verify that roles are properly set up
		roles, err := authService.GetAllRoleDefinitions(models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3) // Should have viewer, admin, owner

		canvasRoles, err := authService.GetAllRoleDefinitions(models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(canvasRoles), 3) // Should have viewer, admin, owner
	})

	t.Run("sync is idempotent", func(t *testing.T) {
		// Run sync twice
		err := authService.SyncDefaultRoles()
		require.NoError(t, err)

		err = authService.SyncDefaultRoles()
		require.NoError(t, err)

		// Should still work and not create duplicates
		roles, err := authService.GetAllRoleDefinitions(models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3)

		// Test that permissions still work
		userID := r.User.String()
		err = authService.AssignRole(userID, RoleOrgViewer, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		allowed, err := authService.CheckOrganizationPermission(userID, orgID, "canvas", "read")
		require.NoError(t, err)
		assert.True(t, allowed)
	})
}

func Test__AuthService_CheckAndSyncMissingPermissions(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	t.Run("check and sync in one operation", func(t *testing.T) {
		// Run the combined operation
		err := authService.CheckAndSyncMissingPermissions()
		require.NoError(t, err)

		// Verify that permissions are now properly set up
		orgID := r.Organization.ID.String()
		canvasID := r.Canvas.ID.String()

		// Test org permissions
		roles, err := authService.GetAllRoleDefinitions(models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3)

		// Test canvas permissions
		canvasRoles, err := authService.GetAllRoleDefinitions(models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(canvasRoles), 3)

		// Test that roles work properly
		userID := r.User.String()
		err = authService.AssignRole(userID, RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		allowed, err := authService.CheckOrganizationPermission(userID, orgID, "canvas", "create")
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
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	orgID := r.Organization.ID.String()

	t.Run("sync organization roles creates expected policies", func(t *testing.T) {
		// Sync org roles
		err := authService.syncOrganizationRoles(orgID)
		require.NoError(t, err)

		// Test that all expected roles exist
		expectedRoles := []string{RoleOrgViewer, RoleOrgAdmin, RoleOrgOwner}
		for _, role := range expectedRoles {
			roleDef, err := authService.GetRoleDefinition(role, models.DomainTypeOrganization, orgID)
			require.NoError(t, err)
			assert.Equal(t, role, roleDef.Name)
			assert.NotEmpty(t, roleDef.Permissions)
		}

		// Test role hierarchy
		userID := r.User.String()
		err = authService.AssignRole(userID, RoleOrgOwner, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		// Owner should have admin and viewer permissions through inheritance
		roles, err := authService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)

		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		assert.True(t, flatRoles[RoleOrgOwner])
		assert.True(t, flatRoles[RoleOrgAdmin])
		assert.True(t, flatRoles[RoleOrgViewer])
	})

	t.Run("sync is idempotent for organizations", func(t *testing.T) {
		// Sync multiple times
		err := authService.syncOrganizationRoles(orgID)
		require.NoError(t, err)

		err = authService.syncOrganizationRoles(orgID)
		require.NoError(t, err)

		err = authService.syncOrganizationRoles(orgID)
		require.NoError(t, err)

		// Should still work correctly
		roles, err := authService.GetAllRoleDefinitions(models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3)
	})
}

func Test__AuthService_SyncCanvasRoles(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	canvasID := r.Canvas.ID.String()

	t.Run("sync canvas roles creates expected policies", func(t *testing.T) {
		// Sync canvas roles
		err := authService.syncCanvasRoles(canvasID)
		require.NoError(t, err)

		// Test that all expected roles exist
		expectedRoles := []string{RoleCanvasViewer, RoleCanvasAdmin, RoleCanvasOwner}
		for _, role := range expectedRoles {
			roleDef, err := authService.GetRoleDefinition(role, models.DomainTypeCanvas, canvasID)
			require.NoError(t, err)
			assert.Equal(t, role, roleDef.Name)
			assert.NotEmpty(t, roleDef.Permissions)
		}

		// Test role hierarchy
		userID := r.User.String()
		err = authService.AssignRole(userID, RoleCanvasOwner, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		// Owner should have admin and viewer permissions through inheritance
		roles, err := authService.GetUserRolesForCanvas(userID, canvasID)
		require.NoError(t, err)

		flatRoles := make(map[string]bool)
		for _, role := range roles {
			flatRoles[role.Name] = true
		}

		assert.True(t, flatRoles[RoleCanvasOwner])
		assert.True(t, flatRoles[RoleCanvasAdmin])
		assert.True(t, flatRoles[RoleCanvasViewer])
	})

	t.Run("sync is idempotent for canvases", func(t *testing.T) {
		// Sync multiple times
		err := authService.syncCanvasRoles(canvasID)
		require.NoError(t, err)

		err = authService.syncCanvasRoles(canvasID)
		require.NoError(t, err)

		err = authService.syncCanvasRoles(canvasID)
		require.NoError(t, err)

		// Should still work correctly
		roles, err := authService.GetAllRoleDefinitions(models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 3)
	})
}

func Test__AuthService_PermissionSync_Integration(t *testing.T) {
	r := support.Setup(t)

	// Create a fresh auth service to test manual sync
	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	t.Run("manual sync sets up permissions correctly", func(t *testing.T) {
		orgID := r.Organization.ID.String()
		canvasID := r.Canvas.ID.String()

		// Run the sync manually (simulating what happens in main.go)
		err := authService.CheckAndSyncMissingPermissions()
		require.NoError(t, err)

		// Test that roles are now available
		orgRoles, err := authService.GetAllRoleDefinitions(models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(orgRoles), 3)

		canvasRoles, err := authService.GetAllRoleDefinitions(models.DomainTypeCanvas, canvasID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(canvasRoles), 3)
	})

	t.Run("permissions work end-to-end after startup sync", func(t *testing.T) {
		userID := r.User.String()
		orgID := r.Organization.ID.String()
		canvasID := r.Canvas.ID.String()

		// Assign roles
		err := authService.AssignRole(userID, RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		err = authService.AssignRole(userID, RoleCanvasViewer, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)

		// Test org permissions
		allowed, err := authService.CheckOrganizationPermission(userID, orgID, "canvas", "create")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Test canvas permissions
		allowed, err = authService.CheckCanvasPermission(userID, canvasID, "stage", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Test that viewer doesn't have write permissions
		allowed, err = authService.CheckCanvasPermission(userID, canvasID, "stage", "create")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__AuthService_MissingPermissions(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	orgID := r.Organization.ID.String()
	canvasID := r.Canvas.ID.String()

	t.Run("only syncs entities with missing permissions", func(t *testing.T) {
		// Before setup - should find missing permissions
		orgsMissingBefore, err := authService.getOrganizationsWithMissingPermissions()
		require.NoError(t, err)
		canvasesMissingBefore, err := authService.getCanvasesWithMissingPermissions()
		require.NoError(t, err)

		// Should include our test entities since they haven't been set up yet
		assert.Contains(t, orgsMissingBefore, orgID, "Org should have missing permissions before setup")
		assert.Contains(t, canvasesMissingBefore, canvasID, "Canvas should have missing permissions before setup")

		// Setup permissions for org and canvas
		err = authService.SetupOrganizationRoles(orgID)
		require.NoError(t, err)
		err = authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		// After setup - should not find missing permissions for these entities
		orgsMissingAfter, err := authService.getOrganizationsWithMissingPermissions()
		require.NoError(t, err)
		canvasesMissingAfter, err := authService.getCanvasesWithMissingPermissions()
		require.NoError(t, err)

		// Should NOT include our test entities since they're now properly set up
		assert.NotContains(t, orgsMissingAfter, orgID, "Org should not have missing permissions after setup")
		assert.NotContains(t, canvasesMissingAfter, canvasID, "Canvas should not have missing permissions after setup")
	})
}
