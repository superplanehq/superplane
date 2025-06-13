package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func Test_ListUsersForRole(t *testing.T) {
	r := support.Setup(t)
	authService := setupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	canvasID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)
	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	// Assign roles to user
	err = authService.AssignRole(r.User.String(), authorization.RoleOrgAdmin, orgID, authorization.DomainOrg)
	require.NoError(t, err)
	err = authService.AssignRole(r.User.String(), authorization.RoleCanvasAdmin, canvasID, authorization.DomainCanvas)
	require.NoError(t, err)

	t.Run("successful list organization users for role", func(t *testing.T) {
		req := &pb.ListOrganizationUsersForRoleRequest{
			OrgId: orgID,
			Role:  authorization.RoleOrgAdmin,
		}

		resp, err := ListOrganizationUsersForRole(ctx, req, authService)
		require.NoError(t, err)
		assert.Contains(t, resp.UserIds, r.User.String())
		assert.Len(t, resp.UserIds, 1)
	})

	t.Run("successful list canvas users for role", func(t *testing.T) {
		req := &pb.ListCanvasUsersForRoleRequest{
			CanvasId: canvasID,
			Role:     authorization.RoleCanvasAdmin,
		}

		resp, err := ListCanvasUsersForRole(ctx, req, authService)
		require.NoError(t, err)
		assert.Contains(t, resp.UserIds, r.User.String())
	})
}
