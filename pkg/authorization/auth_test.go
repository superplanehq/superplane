package authorization

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		err := authService.AssignRole(userID, RoleCanvasOwner, canvasID, DomainCanvas)
		require.NoError(t, err)

		roles, err := authService.GetUserRolesForCanvas(userID, canvasID)
		require.NoError(t, err)
		assert.Equal(t, []string{RoleCanvasOwner, RoleCanvasAdmin, RoleCanvasViewer}, roles)

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
		err := authService.AssignRole(viewerID, RoleCanvasViewer, canvasID, DomainCanvas)
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
		err := authService.AssignRole(adminID, RoleCanvasAdmin, canvasID, DomainCanvas)
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
		err := authService.AssignRole(userID, RoleOrgOwner, orgID, DomainOrg)
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
		allowed, err = authService.CheckOrganizationPermission(userID, orgID, "org", "update")
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = authService.CheckOrganizationPermission(userID, orgID, "org", "delete")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("org admin has limited permissions", func(t *testing.T) {
		adminID := uuid.New().String()
		err := authService.AssignRole(adminID, RoleOrgAdmin, orgID, DomainOrg)
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
		allowed, err = authService.CheckOrganizationPermission(adminID, orgID, "org", "update")
		require.NoError(t, err)
		assert.False(t, allowed)

		allowed, err = authService.CheckOrganizationPermission(adminID, orgID, "org", "delete")
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("org viewer has only read permissions", func(t *testing.T) {
		viewerID := uuid.New().String()
		err := authService.AssignRole(viewerID, RoleOrgViewer, orgID, DomainOrg)
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
		err := authService.AssignRole(userID, RoleOrgAdmin, orgID, DomainOrg)
		require.NoError(t, err)

		// Verify role assignment
		roles, err := authService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)
		assert.Contains(t, roles, RoleOrgAdmin)

		// Remove role
		err = authService.RemoveRole(userID, RoleOrgAdmin, orgID, DomainOrg)
		require.NoError(t, err)

		// Verify role removal
		roles, err = authService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)
		assert.NotContains(t, roles, RoleOrgAdmin)
	})

	t.Run("get users for role", func(t *testing.T) {
		user1 := uuid.New().String()
		user2 := uuid.New().String()

		err := authService.AssignRole(user1, RoleCanvasViewer, canvasID, DomainCanvas)
		require.NoError(t, err)
		err = authService.AssignRole(user2, RoleCanvasViewer, canvasID, DomainCanvas)
		require.NoError(t, err)

		users, err := authService.GetCanvasUsersForRole(RoleCanvasViewer, canvasID)
		require.NoError(t, err)
		assert.Contains(t, users, user1)
		assert.Contains(t, users, user2)
	})

	t.Run("invalid role assignment", func(t *testing.T) {
		err := authService.AssignRole(userID, "invalid_role", orgID, DomainOrg)
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
		err := authService.CreateGroup(orgID, groupName, RoleOrgAdmin)
		require.NoError(t, err)

		// Add users to group
		user1 := uuid.New().String()
		user2 := uuid.New().String()

		err = authService.AddUserToGroup(orgID, user1, groupName)
		require.NoError(t, err)
		err = authService.AddUserToGroup(orgID, user2, groupName)
		require.NoError(t, err)

		// Get group users
		users, err := authService.GetGroupUsers(orgID, groupName)
		require.NoError(t, err)
		assert.Contains(t, users, user1)
		assert.Contains(t, users, user2)

		// Check permissions through group
		allowed, err := authService.CheckOrganizationPermission(user1, orgID, "canvas", "create")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Remove user from group
		err = authService.RemoveUserFromGroup(orgID, user1, groupName)
		require.NoError(t, err)

		// Verify removal
		users, err = authService.GetGroupUsers(orgID, groupName)
		require.NoError(t, err)
		assert.NotContains(t, users, user1)
		assert.Contains(t, users, user2)
	})

	t.Run("create group with invalid role", func(t *testing.T) {
		err := authService.CreateGroup(orgID, "test-group", "invalid_role")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role")
	})

	t.Run("add user to non-existent group", func(t *testing.T) {
		userID := uuid.New().String()
		err := authService.AddUserToGroup(orgID, userID, "non-existent-group")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("get groups and roles", func(t *testing.T) {
		// Create multiple groups
		err := authService.CreateGroup(orgID, "admins", RoleOrgAdmin)
		require.NoError(t, err)
		err = authService.CreateGroup(orgID, "viewers", RoleOrgViewer)
		require.NoError(t, err)

		// Add users to make groups detectable
		user1 := uuid.New().String()
		user2 := uuid.New().String()
		err = authService.AddUserToGroup(orgID, user1, "admins")
		require.NoError(t, err)
		err = authService.AddUserToGroup(orgID, user2, "viewers")
		require.NoError(t, err)

		// Get all groups
		groups, err := authService.GetGroups(orgID)
		require.NoError(t, err)
		assert.Contains(t, groups, "admins")
		assert.Contains(t, groups, "viewers")

		// Get group roles
		roles, err := authService.GetGroupRoles(orgID, "admins")
		require.NoError(t, err)
		assert.Contains(t, roles, RoleOrgAdmin)
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
		err := authService.AssignRole(userID, RoleOrgViewer, org1, DomainOrg)
		require.NoError(t, err)
		err = authService.AssignRole(userID, RoleOrgAdmin, org2, DomainOrg)
		require.NoError(t, err)

		// Get accessible orgs
		orgs, err := authService.GetAccessibleOrgsForUser(userID)
		require.NoError(t, err)
		assert.Contains(t, orgs, org1)
		assert.Contains(t, orgs, org2)
	})

	t.Run("get accessible canvases", func(t *testing.T) {
		// Assign user to canvases
		err := authService.AssignRole(userID, RoleCanvasViewer, canvas1, DomainCanvas)
		require.NoError(t, err)
		err = authService.AssignRole(userID, RoleCanvasOwner, canvas2, DomainCanvas)
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
		allowed, err := authService.CheckOrganizationPermission(userID, orgID, "org", "update")
		require.NoError(t, err)
		assert.True(t, allowed)

		roles, err := authService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)
		assert.Contains(t, roles, RoleOrgOwner)
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
		err := authService.AssignRole(userID, RoleCanvasOwner, canvasID, DomainCanvas)
		require.NoError(t, err)

		// Get implicit roles (should include inherited roles)
		roles, err := authService.GetUserRolesForCanvas(userID, canvasID)
		require.NoError(t, err)

		// Should have all three roles due to hierarchy
		assert.Contains(t, roles, RoleCanvasOwner)
		assert.Contains(t, roles, RoleCanvasAdmin)
		assert.Contains(t, roles, RoleCanvasViewer)
	})

	t.Run("canvas admin inherits viewer permissions", func(t *testing.T) {
		adminID := uuid.New().String()
		err := authService.AssignRole(adminID, RoleCanvasAdmin, canvasID, DomainCanvas)
		require.NoError(t, err)

		roles, err := authService.GetUserRolesForCanvas(adminID, canvasID)
		require.NoError(t, err)

		// Should have admin and viewer roles
		assert.Contains(t, roles, RoleCanvasAdmin)
		assert.Contains(t, roles, RoleCanvasViewer)
		// Should not have owner role
		assert.NotContains(t, roles, RoleCanvasOwner)
	})

	t.Run("org owner inherits admin and viewer permissions", func(t *testing.T) {
		ownerID := uuid.New().String()
		err := authService.AssignRole(ownerID, RoleOrgOwner, orgID, DomainOrg)
		require.NoError(t, err)

		roles, err := authService.GetUserRolesForOrg(ownerID, orgID)
		require.NoError(t, err)

		// Should have all three roles due to hierarchy
		assert.Contains(t, roles, RoleOrgOwner)
		assert.Contains(t, roles, RoleOrgAdmin)
		assert.Contains(t, roles, RoleOrgViewer)
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
		err := authService.AssignRole(userID, RoleOrgAdmin, orgID, DomainOrg)
		require.NoError(t, err)

		// Duplicate assignment should not error
		err = authService.AssignRole(userID, RoleOrgAdmin, orgID, DomainOrg)
		require.NoError(t, err)

		// Should still have the role only once
		roles, err := authService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)
		assert.Contains(t, roles, RoleOrgAdmin)
	})

	t.Run("duplicate group creation fails", func(t *testing.T) {
		groupName := "duplicate-test-group"

		// First creation
		err := authService.CreateGroup(orgID, groupName, RoleOrgViewer)
		require.NoError(t, err)

		// Duplicate creation should fail
		err = authService.CreateGroup(orgID, groupName, RoleOrgViewer)
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
		err = authService.AssignRole(userID, RoleOrgOwner, orgID, DomainOrg)
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
		err = authService.AssignRole(userID, RoleCanvasOwner, canvasID, DomainCanvas)
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
		err := authService.AssignRole(viewerID, RoleCanvasViewer, canvasID, DomainCanvas)
		require.NoError(t, err)
		err = authService.AssignRole(adminID, RoleCanvasAdmin, canvasID, DomainCanvas)
		require.NoError(t, err)
		err = authService.AssignRole(ownerID, RoleCanvasOwner, canvasID, DomainCanvas)
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
		err = authService.AssignRole(viewerID, RoleOrgViewer, orgID, DomainOrg)
		require.NoError(t, err)
		err = authService.AssignRole(adminID, RoleOrgAdmin, orgID, DomainOrg)
		require.NoError(t, err)
		err = authService.AssignRole(ownerID, RoleOrgOwner, orgID, DomainOrg)
		require.NoError(t, err)

		// Check org update permission
		allowed, err := authService.CheckOrganizationPermission(viewerID, orgID, "org", "update")
		require.NoError(t, err)
		assert.False(t, allowed, "Viewer should not have org update permission")

		allowed, err = authService.CheckOrganizationPermission(adminID, orgID, "org", "update")
		require.NoError(t, err)
		assert.False(t, allowed, "Admin should not have org update permission")

		allowed, err = authService.CheckOrganizationPermission(ownerID, orgID, "org", "update")
		require.NoError(t, err)
		assert.True(t, allowed, "Owner should have org update permission")

		// Check org delete permission
		allowed, err = authService.CheckOrganizationPermission(viewerID, orgID, "org", "delete")
		require.NoError(t, err)
		assert.False(t, allowed, "Viewer should not have org delete permission")

		allowed, err = authService.CheckOrganizationPermission(adminID, orgID, "org", "delete")
		require.NoError(t, err)
		assert.False(t, allowed, "Admin should not have org delete permission")

		allowed, err = authService.CheckOrganizationPermission(ownerID, orgID, "org", "delete")
		require.NoError(t, err)
		assert.True(t, allowed, "Owner should have org delete permission")
	})
}
