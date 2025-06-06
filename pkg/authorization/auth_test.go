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

		allowedCanvas, err := authService.CheckCanvasPermission(userID, canvasID, "canvas", "read")
		require.NoError(t, err)
		assert.False(t, allowedCanvas)
	})

	t.Run("canvas owner has all permissions", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleCanvasOwner, canvasID, DomainCanvas)
		require.NoError(t, err)

		// Test all actions
		resources := []string{"eventsource", "stage", "stageevent"}
		actions := []string{"create", "read", "update", "delete"}
		stageEventActions := []string{"approve", "read"}
		memberActions := []string{"invite", "remove"}

		roles, err := authService.GetUserRolesForCanvas(userID, canvasID)
		require.NoError(t, err)
		assert.Equal(t, []string{RoleCanvasOwner, RoleCanvasAdmin, RoleCanvasViewer}, roles)

		for _, resource := range resources {
			if resource == "stageevent" {
				for _, action := range stageEventActions {
					allowed, err := authService.CheckCanvasPermission(userID, canvasID, resource, action)
					require.NoError(t, err)
					assert.True(t, allowed, "Canvas owner should have %s permission for %s", action, resource)
				}
				continue
			}

			if resource == "member" {
				for _, action := range memberActions {
					allowed, err := authService.CheckCanvasPermission(userID, canvasID, resource, action)
					require.NoError(t, err)
					assert.True(t, allowed, "Canvas owner should have %s permission for %s", action, resource)
				}
				continue
			}

			for _, action := range actions {
				allowed, err := authService.CheckCanvasPermission(userID, canvasID, resource, action)
				require.NoError(t, err)
				assert.True(t, allowed, "Canvas owner should have %s permission for %s", action, resource)
			}
		}
	})

	t.Run("canvas viewer has only read permissions", func(t *testing.T) {
		viewerID := uuid.New().String()
		err := authService.AssignRole(viewerID, RoleCanvasViewer, canvasID, DomainCanvas)
		require.NoError(t, err)

		// Should have read permission
		allowed, err := authService.CheckCanvasPermission(viewerID, canvasID, "stageevent", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		// Should not have write permission
		allowed, err = authService.CheckCanvasPermission(viewerID, canvasID, "stageevent", "write")
		require.NoError(t, err)
		assert.False(t, allowed)

		// Should not have admin permission
		allowed, err = authService.CheckCanvasPermission(viewerID, canvasID, "stageevent", "admin")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}
