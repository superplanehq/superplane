package authorization

import (
	"testing"
	"time"

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

	userID := r.User.String()
	canvasID := r.Canvas.ID.String()

	t.Run("user without roles has no permissions", func(t *testing.T) {
		allowed, err := authService.CheckCanvasPermission(userID, canvasID, ActionRead)
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("canvas owner has all permissions", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleCanvasOwner, canvasID)
		require.NoError(t, err)

		// Test all actions
		actions := []string{ActionRead, ActionWrite, ActionCreate, ActionDelete, ActionAdmin}
		for _, action := range actions {
			allowed, err := authService.CheckCanvasPermission(userID, canvasID, action)
			require.NoError(t, err)
			assert.True(t, allowed, "Canvas owner should have %s permission", action)
		}
	})

	t.Run("canvas viewer has only read permissions", func(t *testing.T) {
		viewerID := uuid.New().String()
		err := authService.AssignRole(viewerID, RoleCanvasViewer, canvasID)
		require.NoError(t, err)

		// Should have read permission
		allowed, err := authService.CheckCanvasPermission(viewerID, canvasID, ActionRead)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should not have write permission
		allowed, err = authService.CheckCanvasPermission(viewerID, canvasID, ActionWrite)
		require.NoError(t, err)
		assert.False(t, allowed)

		// Should not have admin permission
		allowed, err = authService.CheckCanvasPermission(viewerID, canvasID, ActionAdmin)
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__AuthService_RoleHierarchy(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)

	userID := r.User.String()
	canvasID := r.Canvas.ID.String()

	t.Run("canvas admin inherits developer permissions", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleCanvasAdmin, canvasID)
		require.NoError(t, err)

		// Admin should have read (from viewer)
		allowed, err := authService.CheckCanvasPermission(userID, canvasID, ActionRead)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Admin should have create (from contributor/developer)
		allowed, err = authService.CheckCanvasPermission(userID, canvasID, ActionCreate)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Admin should have write (from admin)
		allowed, err = authService.CheckCanvasPermission(userID, canvasID, ActionWrite)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("canvas developer inherits contributor permissions", func(t *testing.T) {
		developerID := uuid.New().String()
		err := authService.AssignRole(developerID, RoleCanvasDeveloper, canvasID)
		require.NoError(t, err)

		// Developer should have read (from viewer)
		allowed, err := authService.CheckCanvasPermission(developerID, canvasID, ActionRead)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Developer should have create (from contributor)
		allowed, err = authService.CheckCanvasPermission(developerID, canvasID, ActionCreate)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Developer should have write (from developer)
		allowed, err = authService.CheckCanvasPermission(developerID, canvasID, ActionWrite)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Developer should NOT have admin permission
		allowed, err = authService.CheckCanvasPermission(developerID, canvasID, ActionAdmin)
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__AuthService_WithExistingCanvas(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)

	userID := r.User.String()
	canvasID := r.Canvas.ID.String()

	t.Run("assign user as canvas owner", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleCanvasOwner, canvasID)
		require.NoError(t, err)

		// Verify user can perform admin actions on this canvas
		allowed, err := authService.CheckCanvasPermission(userID, canvasID, ActionAdmin)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Verify user is listed as canvas owner
		canvasUsers, err := authService.GetCanvasUsers(canvasID)
		require.NoError(t, err)
		assert.Contains(t, canvasUsers[RoleCanvasOwner], userID)
	})

	t.Run("user can access canvas resources", func(t *testing.T) {
		// Test stage access
		stageResource := "stage"
		allowed, err := authService.CheckPermission(userID, stageResource, ActionRead)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Test execution access
		executionResource := "execution"
		allowed, err = authService.CheckPermission(userID, executionResource, ActionRead)
		require.NoError(t, err)
		assert.True(t, allowed)
	})
}

func Test__AuthService_WithExistingSource(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source: true,
	})

	authService, err := NewAuthService()
	require.NoError(t, err)

	userID := r.User.String()
	canvasID := r.Canvas.ID.String()

	t.Run("canvas developer can access event sources", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleCanvasDeveloper, canvasID)
		require.NoError(t, err)

		// Should be able to read event sources
		allowed, err := authService.CheckPermission(userID, ResourceEventSource, ActionRead)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("canvas viewer cannot modify event sources", func(t *testing.T) {
		viewerID := uuid.New().String()
		err := authService.AssignRole(viewerID, RoleCanvasViewer, canvasID)
		require.NoError(t, err)

		allowed, err := authService.CheckPermission(viewerID, ResourceEventSource, ActionWrite)
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__AuthService_RoleManagement(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)

	userID := r.User.String()
	canvasID := r.Canvas.ID.String()

	t.Run("assign and retrieve user roles", func(t *testing.T) {
		// Assign canvas owner role
		err := authService.AssignRole(userID, RoleCanvasOwner, canvasID)
		require.NoError(t, err)

		// Check roles
		roles, err := authService.GetUserRoles(userID)
		require.NoError(t, err)
		assert.Contains(t, roles, RoleCanvasOwner+":"+canvasID)
	})

	t.Run("remove user roles", func(t *testing.T) {
		// Assign role first
		err := authService.AssignRole(userID, RoleCanvasViewer, canvasID)
		require.NoError(t, err)

		// Verify role exists
		roles, err := authService.GetUserRoles(userID)
		require.NoError(t, err)
		assert.Contains(t, roles, RoleCanvasViewer+":"+canvasID)

		// Remove role
		err = authService.RemoveRole(userID, RoleCanvasViewer, canvasID)
		require.NoError(t, err)

		// Verify role removed
		roles, err = authService.GetUserRoles(userID)
		require.NoError(t, err)
		assert.NotContains(t, roles, RoleCanvasViewer+":"+canvasID)
	})

	t.Run("get users for role", func(t *testing.T) {
		user1 := uuid.New().String()
		user2 := uuid.New().String()
		role := RoleCanvasDeveloper + ":" + canvasID

		// Assign same role to multiple users
		err := authService.AssignRole(user1, RoleCanvasDeveloper, canvasID)
		require.NoError(t, err)
		err = authService.AssignRole(user2, RoleCanvasDeveloper, canvasID)
		require.NoError(t, err)

		// Get users for role
		users, err := authService.GetUsersForRole(role)
		require.NoError(t, err)
		assert.Contains(t, users, user1)
		assert.Contains(t, users, user2)
	})
}

func Test__AuthService_CanvasUsers(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)

	canvasID := r.Canvas.ID.String()

	t.Run("get canvas users with different roles", func(t *testing.T) {
		// Assign different users to different roles
		err := authService.AssignRole("owner1", RoleCanvasOwner, canvasID)
		require.NoError(t, err)
		err = authService.AssignRole("admin1", RoleCanvasAdmin, canvasID)
		require.NoError(t, err)
		err = authService.AssignRole("dev1", RoleCanvasDeveloper, canvasID)
		require.NoError(t, err)
		err = authService.AssignRole("dev2", RoleCanvasDeveloper, canvasID)
		require.NoError(t, err)
		err = authService.AssignRole("viewer1", RoleCanvasViewer, canvasID)
		require.NoError(t, err)

		// Get canvas users
		canvasUsers, err := authService.GetCanvasUsers(canvasID)
		require.NoError(t, err)

		// Verify users are categorized by role
		assert.Contains(t, canvasUsers[RoleCanvasOwner], "owner1")
		assert.Contains(t, canvasUsers[RoleCanvasAdmin], "admin1")
		assert.Contains(t, canvasUsers[RoleCanvasDeveloper], "dev1")
		assert.Contains(t, canvasUsers[RoleCanvasDeveloper], "dev2")
		assert.Contains(t, canvasUsers[RoleCanvasViewer], "viewer1")
		assert.Len(t, canvasUsers[RoleCanvasDeveloper], 2)
	})
}

func Test__AuthService_InviteUserToCanvas(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)

	userID := uuid.New().String()
	canvasID := r.Canvas.ID.String()

	t.Run("invite user with valid role", func(t *testing.T) {
		err := authService.InviteUserToCanvas(userID, canvasID, RoleCanvasDeveloper)
		require.NoError(t, err)

		// Verify user has the role
		allowed, err := authService.CheckCanvasPermission(userID, canvasID, ActionWrite)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("invite user with invalid role fails", func(t *testing.T) {
		err := authService.InviteUserToCanvas(userID, canvasID, "invalid:role")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid canvas role")
	})
}

func Test__AuthService_CreateOrganizationOwner(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)

	userID := r.User.String()
	orgID := uuid.New().String()

	t.Run("create organization owner", func(t *testing.T) {
		err := authService.CreateOrganizationOwner(userID, orgID)
		require.NoError(t, err)

		// Verify user has organization owner permissions
		allowed, err := authService.CheckOrganizationPermission(userID, orgID, ActionAdmin)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Verify role assignment
		roles, err := authService.GetUserRoles(userID)
		require.NoError(t, err)
		assert.Contains(t, roles, RoleOrgOwner+":"+orgID)
	})
}

func Test__AuthService_MultipleCanvases(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)

	userID := r.User.String()
	canvas1ID := r.Canvas.ID.String()

	// Create a second canvas
	canvas2, err := models.CreateCanvas(r.User, "test-canvas-2")
	require.NoError(t, err)
	canvas2ID := canvas2.ID.String()

	t.Run("user has access only to assigned canvas", func(t *testing.T) {
		// Assign user to canvas1 only
		err := authService.AssignRole(userID, RoleCanvasDeveloper, canvas1ID)
		require.NoError(t, err)

		// Should have access to canvas1
		allowed, err := authService.CheckCanvasPermission(userID, canvas1ID, ActionRead)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should NOT have access to canvas2
		allowed, err = authService.CheckCanvasPermission(userID, canvas2ID, ActionRead)
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("user can have different roles on different canvases", func(t *testing.T) {
		// Assign different roles to different canvases
		err := authService.AssignRole(userID, RoleCanvasOwner, canvas1ID)
		require.NoError(t, err)
		err = authService.AssignRole(userID, RoleCanvasViewer, canvas2ID)
		require.NoError(t, err)

		// Should have admin access to canvas1
		allowed, err := authService.CheckCanvasPermission(userID, canvas1ID, ActionAdmin)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should only have read access to canvas2
		allowed, err = authService.CheckCanvasPermission(userID, canvas2ID, ActionRead)
		require.NoError(t, err)
		assert.True(t, allowed)

		allowed, err = authService.CheckCanvasPermission(userID, canvas2ID, ActionWrite)
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__AuthService_StageAndExecutionAccess(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source: true,
	})

	authService, err := NewAuthService()
	require.NoError(t, err)

	// Create a stage using existing test infrastructure
	err = r.Canvas.CreateStage("test-stage", r.User.String(), []models.StageCondition{}, support.ExecutorSpec(), []models.StageConnection{
		{
			SourceID:   r.Source.ID,
			SourceType: models.SourceTypeEventSource,
		},
	}, []models.InputDefinition{}, []models.InputMapping{}, []models.OutputDefinition{}, []models.ValueDefinition{})
	require.NoError(t, err)

	stage, err := r.Canvas.FindStageByName("test-stage")
	require.NoError(t, err)

	userID := r.User.String()
	canvasID := r.Canvas.ID.String()

	t.Run("canvas developer can access stage and create executions", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleCanvasDeveloper, canvasID)
		require.NoError(t, err)

		// Should be able to read stages
		allowed, err := authService.CheckPermission(userID, ResourceStage, ActionRead)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should be able to write stages
		allowed, err = authService.CheckPermission(userID, ResourceStage, ActionWrite)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should be able to create executions
		allowed, err = authService.CheckPermission(userID, ResourceExecution, ActionCreate)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("canvas viewer can only read stages and executions", func(t *testing.T) {
		viewerID := uuid.New().String()
		err := authService.AssignRole(viewerID, RoleCanvasViewer, canvasID)
		require.NoError(t, err)

		// Should be able to read stages
		allowed, err := authService.CheckPermission(viewerID, ResourceStage, ActionRead)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should NOT be able to write stages
		allowed, err = authService.CheckPermission(viewerID, ResourceStage, ActionWrite)
		require.NoError(t, err)
		assert.False(t, allowed)

		// Should NOT be able to create executions
		allowed, err = authService.CheckPermission(viewerID, ResourceExecution, ActionCreate)
		require.NoError(t, err)
		assert.False(t, allowed)

		// Create an execution to test read access
		execution := support.CreateExecution(t, r.Source, stage)

		// Should be able to read executions
		allowed, err = authService.CheckPermission(viewerID, ResourceExecution, ActionRead)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Verify execution was created
		assert.NotNil(t, execution)
	})
}

func Test__AuthService_OrganizationLevel(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)

	userID := r.User.String()
	orgID := uuid.New().String()

	t.Run("organization admin can manage canvases", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleOrgAdmin, orgID)
		require.NoError(t, err)

		// Should be able to admin canvases
		allowed, err := authService.CheckPermission(userID, ResourceCanvas, ActionAdmin)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should be able to manage users
		allowed, err = authService.CheckPermission(userID, ResourceUser, ActionWrite)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("organization member has limited access", func(t *testing.T) {
		memberID := uuid.New().String()
		err := authService.AssignRole(memberID, RoleOrgMember, orgID)
		require.NoError(t, err)

		// Should be able to read organization
		allowed, err := authService.CheckPermission(memberID, ResourceOrganization, ActionRead)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should NOT be able to admin organization
		allowed, err = authService.CheckPermission(memberID, ResourceOrganization, ActionAdmin)
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__AuthService_ErrorHandling(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)

	userID := r.User.String()
	canvasID := r.Canvas.ID.String()

	t.Run("check permission for non-existent user", func(t *testing.T) {
		nonExistentUser := uuid.New().String()
		allowed, err := authService.CheckCanvasPermission(nonExistentUser, canvasID, ActionRead)
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("assign role to user multiple times", func(t *testing.T) {
		// Assign role first time
		err := authService.AssignRole(userID, RoleCanvasViewer, canvasID)
		require.NoError(t, err)

		// Assign same role again - should not error
		err = authService.AssignRole(userID, RoleCanvasViewer, canvasID)
		require.NoError(t, err)

		// Should still have the role
		roles, err := authService.GetUserRoles(userID)
		require.NoError(t, err)
		assert.Contains(t, roles, RoleCanvasViewer+":"+canvasID)
	})

	t.Run("remove non-existent role", func(t *testing.T) {
		// Should not error when removing role that doesn't exist
		err := authService.RemoveRole(userID, RoleCanvasOwner, "non-existent-canvas")
		require.NoError(t, err)
	})
}

func Test__AuthService_PermissionPolicies(t *testing.T) {
	support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)

	t.Run("add and remove custom permissions", func(t *testing.T) {
		role := "custom:role"
		resource := "custom:resource"
		action := "custom:action"

		// Add custom permission
		err := authService.AddPermission(role, resource, action)
		require.NoError(t, err)

		// Assign role to user
		userID := "customuser"
		err = authService.AssignRole(userID, role, "")
		require.NoError(t, err)

		// Check permission
		allowed, err := authService.CheckPermission(userID, resource, action)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Remove permission
		err = authService.RemovePermission(role, resource, action)
		require.NoError(t, err)

		// Check permission again
		allowed, err = authService.CheckPermission(userID, resource, action)
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__AuthService_Performance(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)

	canvasID := r.Canvas.ID.String()

	// Setup multiple users with different roles
	const numUsers = 100
	userIDs := make([]string, numUsers)
	for i := 0; i < numUsers; i++ {
		userIDs[i] = uuid.New().String()

		// Assign different roles to users
		var role string
		switch i % 5 {
		case 0:
			role = RoleCanvasOwner
		case 1:
			role = RoleCanvasAdmin
		case 2:
			role = RoleCanvasDeveloper
		case 3:
			role = RoleCanvasContributor
		default:
			role = RoleCanvasViewer
		}

		err := authService.AssignRole(userIDs[i], role, canvasID)
		require.NoError(t, err)
	}

	t.Run("bulk permission checks perform well", func(t *testing.T) {
		start := time.Now()

		for _, userID := range userIDs {
			_, err := authService.CheckCanvasPermission(userID, canvasID, ActionRead)
			require.NoError(t, err)
		}

		duration := time.Since(start)
		// Should complete 100 permission checks in reasonable time
		assert.Less(t, duration.Milliseconds(), int64(1000), "100 permission checks should complete in less than 1 second")
	})

	t.Run("get canvas users performs well", func(t *testing.T) {
		start := time.Now()

		canvasUsers, err := authService.GetCanvasUsers(canvasID)
		require.NoError(t, err)

		duration := time.Since(start)
		assert.Less(t, duration.Milliseconds(), int64(100), "Getting canvas users should complete in less than 100ms")

		// Verify all users are returned
		totalUsers := 0
		for _, users := range canvasUsers {
			totalUsers += len(users)
		}
		assert.Equal(t, numUsers, totalUsers)
	})
}

// Benchmark tests using real database
func BenchmarkAuthService_CheckPermission(b *testing.B) {
	r := support.Setup(&testing.T{})

	authService, err := NewAuthService()
	if err != nil {
		b.Fatal(err)
	}

	userID := r.User.String()
	canvasID := r.Canvas.ID.String()

	// Setup user with role
	err = authService.AssignRole(userID, RoleCanvasDeveloper, canvasID)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := authService.CheckCanvasPermission(userID, canvasID, ActionRead)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAuthService_AssignRole(b *testing.B) {
	r := support.Setup(&testing.T{})

	authService, err := NewAuthService()
	if err != nil {
		b.Fatal(err)
	}

	canvasID := r.Canvas.ID.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := uuid.New().String()
		err := authService.AssignRole(userID, RoleCanvasViewer, canvasID)
		if err != nil {
			b.Fatal(err)
		}
	}
}
