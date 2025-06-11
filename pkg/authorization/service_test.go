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
		err = authService.RemoveRole(userID, RoleOrgAdmin, orgID, DomainOrg)
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
		err := authService.AssignRole(userID, RoleCanvasOwner, canvasID, DomainCanvas)
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
		err := authService.AssignRole(adminID, RoleCanvasAdmin, canvasID, DomainCanvas)
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
		err := authService.AssignRole(ownerID, RoleOrgOwner, orgID, DomainOrg)
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
		err := authService.AssignRole(userID, RoleOrgAdmin, orgID, DomainOrg)
		require.NoError(t, err)

		// Duplicate assignment should not error
		err = authService.AssignRole(userID, RoleOrgAdmin, orgID, DomainOrg)
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
		viewerRole, err := authService.GetRoleDefinition(RoleOrgViewer, DomainOrg, orgID)
		require.NoError(t, err)
		assert.Equal(t, RoleOrgViewer, viewerRole.Name)
		assert.Equal(t, DomainOrg, viewerRole.DomainType)
		assert.NotEmpty(t, viewerRole.Description)
		assert.True(t, viewerRole.Readonly)
		assert.NotEmpty(t, viewerRole.Permissions)

		// Test org admin role
		adminRole, err := authService.GetRoleDefinition(RoleOrgAdmin, DomainOrg, orgID)
		require.NoError(t, err)
		assert.Equal(t, RoleOrgAdmin, adminRole.Name)
		assert.Equal(t, DomainOrg, adminRole.DomainType)
		assert.NotEmpty(t, adminRole.Description)
		assert.True(t, adminRole.Readonly)
		assert.NotEmpty(t, adminRole.Permissions)

		// Test org owner role
		ownerRole, err := authService.GetRoleDefinition(RoleOrgOwner, DomainOrg, orgID)
		require.NoError(t, err)
		assert.Equal(t, RoleOrgOwner, ownerRole.Name)
		assert.Equal(t, DomainOrg, ownerRole.DomainType)
		assert.NotEmpty(t, ownerRole.Description)
		assert.True(t, ownerRole.Readonly)
		assert.NotEmpty(t, ownerRole.Permissions)
	})

	t.Run("get canvas role definition", func(t *testing.T) {
		// Test canvas viewer role
		viewerRole, err := authService.GetRoleDefinition(RoleCanvasViewer, DomainCanvas, canvasID)
		require.NoError(t, err)
		assert.Equal(t, RoleCanvasViewer, viewerRole.Name)
		assert.Equal(t, DomainCanvas, viewerRole.DomainType)
		assert.NotEmpty(t, viewerRole.Description)
		assert.True(t, viewerRole.Readonly)
		assert.NotEmpty(t, viewerRole.Permissions)

		// Test canvas admin role
		adminRole, err := authService.GetRoleDefinition(RoleCanvasAdmin, DomainCanvas, canvasID)
		require.NoError(t, err)
		assert.Equal(t, RoleCanvasAdmin, adminRole.Name)
		assert.Equal(t, DomainCanvas, adminRole.DomainType)
		assert.NotEmpty(t, adminRole.Description)
		assert.True(t, adminRole.Readonly)
		assert.NotEmpty(t, adminRole.Permissions)

		// Test canvas owner role
		ownerRole, err := authService.GetRoleDefinition(RoleCanvasOwner, DomainCanvas, canvasID)
		require.NoError(t, err)
		assert.Equal(t, RoleCanvasOwner, ownerRole.Name)
		assert.Equal(t, DomainCanvas, ownerRole.DomainType)
		assert.NotEmpty(t, ownerRole.Description)
		assert.True(t, ownerRole.Readonly)
		assert.NotEmpty(t, ownerRole.Permissions)
	})

	t.Run("error cases", func(t *testing.T) {
		// Test non-existent role
		_, err := authService.GetRoleDefinition("non_existent_role", DomainOrg, orgID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test non-existent domain
		_, err = authService.GetRoleDefinition(RoleOrgViewer, DomainOrg, "non-existent-org")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test invalid domain type
		_, err = authService.GetRoleDefinition(RoleOrgViewer, "invalid_domain", orgID)
		assert.Error(t, err)
	})

	t.Run("permissions are populated", func(t *testing.T) {
		role, err := authService.GetRoleDefinition(RoleOrgAdmin, DomainOrg, orgID)
		require.NoError(t, err)

		// Check that permissions have all required fields
		for _, perm := range role.Permissions {
			assert.NotEmpty(t, perm.Resource)
			assert.NotEmpty(t, perm.Action)
			assert.Equal(t, DomainOrg, perm.DomainType)
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
		roles, err := authService.GetAllRoleDefinitions(DomainOrg, orgID)
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
			assert.Equal(t, DomainOrg, role.DomainType)
			assert.NotEmpty(t, role.Description)
			assert.True(t, role.Readonly)
			assert.NotEmpty(t, role.Permissions)
		}
	})

	t.Run("get all canvas roles", func(t *testing.T) {
		roles, err := authService.GetAllRoleDefinitions(DomainCanvas, canvasID)
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
			assert.Equal(t, DomainCanvas, role.DomainType)
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
		roles1, err := authService.GetAllRoleDefinitions(DomainOrg, orgID)
		require.NoError(t, err)

		roles2, err := authService.GetAllRoleDefinitions(DomainOrg, anotherOrgID)
		require.NoError(t, err)

		assert.Equal(t, len(roles1), len(roles2))
	})

	t.Run("empty responses", func(t *testing.T) {
		// Test invalid domain type
		definitions, _ := authService.GetAllRoleDefinitions("invalid_domain", orgID)
		assert.Empty(t, definitions)

		// Test non-existent domain
		definitions, _ = authService.GetAllRoleDefinitions(DomainOrg, "non-existent-org")
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
		viewerPermissions, err := authService.GetRolePermissions(RoleOrgViewer, DomainOrg, orgID)
		require.NoError(t, err)
		assert.NotEmpty(t, viewerPermissions)

		// All permissions should be read-only
		for _, perm := range viewerPermissions {
			assert.Equal(t, "read", perm.Action)
			assert.Equal(t, DomainOrg, perm.DomainType)
		}

		// Test org admin permissions (should include viewer permissions + more)
		adminPermissions, err := authService.GetRolePermissions(RoleOrgAdmin, DomainOrg, orgID)
		require.NoError(t, err)
		assert.NotEmpty(t, adminPermissions)
		assert.GreaterOrEqual(t, len(adminPermissions), len(viewerPermissions))

		// Should have various actions
		actions := make(map[string]bool)
		for _, perm := range adminPermissions {
			actions[perm.Action] = true
			assert.Equal(t, DomainOrg, perm.DomainType)
		}
		assert.True(t, actions["read"], "Admin should have read permissions")

		// Test org owner permissions (should include admin permissions + more)
		ownerPermissions, err := authService.GetRolePermissions(RoleOrgOwner, DomainOrg, orgID)
		require.NoError(t, err)
		assert.NotEmpty(t, ownerPermissions)
		assert.GreaterOrEqual(t, len(ownerPermissions), len(adminPermissions))
	})

	t.Run("get canvas role permissions", func(t *testing.T) {
		// Test canvas viewer permissions
		viewerPermissions, err := authService.GetRolePermissions(RoleCanvasViewer, DomainCanvas, canvasID)
		require.NoError(t, err)
		assert.NotEmpty(t, viewerPermissions)

		// All permissions should be read-only
		for _, perm := range viewerPermissions {
			assert.Equal(t, "read", perm.Action)
			assert.Equal(t, DomainCanvas, perm.DomainType)
		}

		// Test canvas admin permissions
		adminPermissions, err := authService.GetRolePermissions(RoleCanvasAdmin, DomainCanvas, canvasID)
		require.NoError(t, err)
		assert.NotEmpty(t, adminPermissions)
		assert.GreaterOrEqual(t, len(adminPermissions), len(viewerPermissions))

		// Test canvas owner permissions
		ownerPermissions, err := authService.GetRolePermissions(RoleCanvasOwner, DomainCanvas, canvasID)
		require.NoError(t, err)
		assert.NotEmpty(t, ownerPermissions)
		assert.GreaterOrEqual(t, len(ownerPermissions), len(adminPermissions))
	})

	t.Run("permissions include inheritance", func(t *testing.T) {
		// Canvas admin should have all viewer permissions plus admin-specific ones
		viewerPermissions, err := authService.GetRolePermissions(RoleCanvasViewer, DomainCanvas, canvasID)
		require.NoError(t, err)

		adminPermissions, err := authService.GetRolePermissions(RoleCanvasAdmin, DomainCanvas, canvasID)
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
		_, err := authService.GetRolePermissions("non_existent_role", DomainOrg, orgID)
		assert.Error(t, err)

		// Test non-existent domain
		_, err = authService.GetRolePermissions(RoleOrgViewer, DomainOrg, "non-existent-org")
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
		viewerHierarchy, err := authService.GetRoleHierarchy(RoleOrgViewer, DomainOrg, orgID)
		require.NoError(t, err)
		assert.Contains(t, viewerHierarchy, RoleOrgViewer)

		// Test org admin hierarchy (should include itself and inherited roles)
		adminHierarchy, err := authService.GetRoleHierarchy(RoleOrgAdmin, DomainOrg, orgID)
		require.NoError(t, err)
		assert.Contains(t, adminHierarchy, RoleOrgAdmin)
		// May also include inherited roles depending on setup

		// Test org owner hierarchy (should include itself and inherited roles)
		ownerHierarchy, err := authService.GetRoleHierarchy(RoleOrgOwner, DomainOrg, orgID)
		require.NoError(t, err)
		assert.Contains(t, ownerHierarchy, RoleOrgOwner)
		// Should be the longest hierarchy
		assert.GreaterOrEqual(t, len(ownerHierarchy), len(adminHierarchy))
	})

	t.Run("get canvas role hierarchy", func(t *testing.T) {
		// Test canvas viewer hierarchy
		viewerHierarchy, err := authService.GetRoleHierarchy(RoleCanvasViewer, DomainCanvas, canvasID)
		require.NoError(t, err)
		assert.Contains(t, viewerHierarchy, RoleCanvasViewer)

		// Test canvas admin hierarchy
		adminHierarchy, err := authService.GetRoleHierarchy(RoleCanvasAdmin, DomainCanvas, canvasID)
		require.NoError(t, err)
		assert.Contains(t, adminHierarchy, RoleCanvasAdmin)

		// Test canvas owner hierarchy
		ownerHierarchy, err := authService.GetRoleHierarchy(RoleCanvasOwner, DomainCanvas, canvasID)
		require.NoError(t, err)
		assert.Contains(t, ownerHierarchy, RoleCanvasOwner)
		// Should be the longest hierarchy
		assert.GreaterOrEqual(t, len(ownerHierarchy), len(adminHierarchy))
	})

	t.Run("hierarchy includes inheritance", func(t *testing.T) {
		// Canvas owner should include admin in hierarchy (if inheritance is set up)
		ownerHierarchy, err := authService.GetRoleHierarchy(RoleCanvasOwner, DomainCanvas, canvasID)
		require.NoError(t, err)

		// The exact inheritance depends on CSV setup, but owner should have most roles
		assert.GreaterOrEqual(t, len(ownerHierarchy), 1) // At least includes itself

		// Admin should have fewer or equal roles than owner
		adminHierarchy, err := authService.GetRoleHierarchy(RoleCanvasAdmin, DomainCanvas, canvasID)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(adminHierarchy), len(ownerHierarchy))
	})

	t.Run("hierarchy is unique", func(t *testing.T) {
		hierarchy, err := authService.GetRoleHierarchy(RoleOrgOwner, DomainOrg, orgID)
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
		_, err := authService.GetRoleHierarchy("non_existent_role", DomainOrg, orgID)
		assert.Error(t, err)

		// Test non-existent domain
		_, err = authService.GetRoleHierarchy(RoleOrgViewer, DomainOrg, "non-existent-org")
		assert.Error(t, err)

		// Test invalid domain type
		_, err = authService.GetRoleHierarchy(RoleOrgViewer, "invalid_domain", orgID)
		assert.Error(t, err)
	})
}
