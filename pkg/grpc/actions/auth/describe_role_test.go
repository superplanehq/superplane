package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test_DescribeRole(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful role description", func(t *testing.T) {
		resp, err := DescribeRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp.Role)
		assert.NotNil(t, resp.Role.Spec.InheritedRole)
		assert.Equal(t, models.RoleOrgAdmin, resp.Role.Metadata.Name)
		assert.Equal(t, models.RoleOrgViewer, resp.Role.Spec.InheritedRole.Metadata.Name)
		assert.Len(t, resp.Role.Spec.Permissions, 25)
		assert.Len(t, resp.Role.Spec.InheritedRole.Spec.Permissions, 2)

		// Test beautiful display names and descriptions
		assert.Equal(t, "Admin", resp.Role.Spec.DisplayName)
		assert.Equal(t, "Viewer", resp.Role.Spec.InheritedRole.Spec.DisplayName)
		assert.Contains(t, resp.Role.Spec.Description, "Can manage canvases, users, groups, and roles")
		assert.Contains(t, resp.Role.Spec.InheritedRole.Spec.Description, "Read-only access to organization resources")
	})

	t.Run("successful canvas role description", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		resp, err := DescribeRole(ctx, models.DomainTypeCanvas, canvasID, models.RoleCanvasAdmin, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp.Role)
		assert.Equal(t, models.RoleCanvasAdmin, resp.Role.Metadata.Name)

		// Test beautiful display names and descriptions for canvas roles
		assert.Equal(t, "Admin", resp.Role.Spec.DisplayName)
		assert.Contains(t, resp.Role.Spec.Description, "Can manage stages, events, connections, and secrets")
	})
}
