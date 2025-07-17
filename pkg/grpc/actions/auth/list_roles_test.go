package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
)

func Test_ListRoles(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful list roles", func(t *testing.T) {
		req := &pb.ListRolesRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
		}

		resp, err := ListRoles(ctx, req, authService)
		require.NoError(t, err)
		assert.Equal(t, len(resp.Roles), 3) // viewer, admin, owner

		// Should have expected roles
		roleNames := make([]string, len(resp.Roles))
		for i, role := range resp.Roles {
			roleNames[i] = role.Name
		}
		assert.Contains(t, roleNames, authorization.RoleOrgViewer)
		assert.Contains(t, roleNames, authorization.RoleOrgAdmin)
		assert.Contains(t, roleNames, authorization.RoleOrgOwner)
		assert.Len(t, resp.Roles, 3)

		// Test beautiful display names and descriptions for each role
		for _, role := range resp.Roles {
			assert.NotEmpty(t, role.DisplayName, "DisplayName should not be empty for role %s", role.Name)
			assert.NotEmpty(t, role.Description, "Description should not be empty for role %s", role.Name)
			
			switch role.Name {
			case authorization.RoleOrgOwner:
				assert.Equal(t, "Owner", role.DisplayName)
				assert.Contains(t, role.Description, "Full control over organization settings")
			case authorization.RoleOrgAdmin:
				assert.Equal(t, "Admin", role.DisplayName)
				assert.Contains(t, role.Description, "Can manage canvases, users, groups, and roles")
			case authorization.RoleOrgViewer:
				assert.Equal(t, "Viewer", role.DisplayName)
				assert.Contains(t, role.Description, "Read-only access to organization resources")
			}
		}
	})

	t.Run("successful list canvas roles", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		req := &pb.ListRolesRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
			DomainId:   canvasID,
		}

		resp, err := ListRoles(ctx, req, authService)
		require.NoError(t, err)
		assert.Equal(t, len(resp.Roles), 3) // viewer, admin, owner

		// Test beautiful display names and descriptions for canvas roles
		for _, role := range resp.Roles {
			assert.NotEmpty(t, role.DisplayName, "DisplayName should not be empty for role %s", role.Name)
			assert.NotEmpty(t, role.Description, "Description should not be empty for role %s", role.Name)
			
			switch role.Name {
			case authorization.RoleCanvasOwner:
				assert.Equal(t, "Owner", role.DisplayName)
				assert.Contains(t, role.Description, "Full control over canvas settings")
			case authorization.RoleCanvasAdmin:
				assert.Equal(t, "Admin", role.DisplayName)
				assert.Contains(t, role.Description, "Can manage stages, events, connections, and secrets")
			case authorization.RoleCanvasViewer:
				assert.Equal(t, "Viewer", role.DisplayName)
				assert.Contains(t, role.Description, "Read-only access to canvas resources")
			}
		}
	})

	t.Run("invalid request - unspecified domain type", func(t *testing.T) {
		req := &pb.ListRolesRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainId:   orgID,
		}

		_, err := ListRoles(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})
}
