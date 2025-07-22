package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
)

func Test_DescribeRole(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful role description", func(t *testing.T) {
		req := &pb.DescribeRoleRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			Role:       models.RoleOrgAdmin,
		}

		resp, err := DescribeRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp.Role)
		assert.NotNil(t, resp.Role.InheritedRole)
		assert.Equal(t, models.RoleOrgAdmin, resp.Role.Name)
		assert.Equal(t, models.RoleOrgViewer, resp.Role.InheritedRole.Name)
		assert.Len(t, resp.Role.Permissions, 18)
		assert.Len(t, resp.Role.InheritedRole.Permissions, 2)

		// Test beautiful display names and descriptions
		assert.Equal(t, "Admin", resp.Role.DisplayName)
		assert.Equal(t, "Viewer", resp.Role.InheritedRole.DisplayName)
		assert.Contains(t, resp.Role.Description, "Can manage canvases, users, groups, and roles")
		assert.Contains(t, resp.Role.InheritedRole.Description, "Read-only access to organization resources")
	})

	t.Run("successful canvas role description", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		req := &pb.DescribeRoleRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
			DomainId:   canvasID,
			Role:       models.RoleCanvasAdmin,
		}

		resp, err := DescribeRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp.Role)
		assert.Equal(t, models.RoleCanvasAdmin, resp.Role.Name)

		// Test beautiful display names and descriptions for canvas roles
		assert.Equal(t, "Admin", resp.Role.DisplayName)
		assert.Contains(t, resp.Role.Description, "Can manage stages, events, connections, and secrets")
	})

	t.Run("invalid request - missing domain ID", func(t *testing.T) {
		req := &pb.DescribeRoleRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   "",
			Role:       models.RoleOrgAdmin,
		}

		_, err := DescribeRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain ID must be specified")
	})
}
