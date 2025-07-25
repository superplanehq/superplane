package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
)

func TestUpdateGroup(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful role update", func(t *testing.T) {
		err := authService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group", models.RoleOrgViewer, "Test Group", "Test Description")
		require.NoError(t, err)

		groupSpec := &pb.Group_Spec{
			Role: models.RoleOrgAdmin,
		}

		resp, err := UpdateGroup(ctx, models.DomainTypeOrganization, orgID, "test-group", groupSpec, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, models.RoleOrgAdmin, resp.Group.Spec.Role)

		role, err := authService.GetGroupRole(orgID, models.DomainTypeOrganization, "test-group")
		require.NoError(t, err)
		assert.Equal(t, models.RoleOrgAdmin, role)
	})

	t.Run("successful metadata update", func(t *testing.T) {
		err := authService.CreateGroup(orgID, models.DomainTypeOrganization, "metadata-group", models.RoleOrgViewer, "Metadata Group", "Metadata Description")
		require.NoError(t, err)

		groupSpec := &pb.Group_Spec{
			DisplayName: "Updated Display Name",
			Description: "Updated Description",
		}

		resp, err := UpdateGroup(ctx, models.DomainTypeOrganization, orgID, "metadata-group", groupSpec, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "Updated Display Name", resp.Group.Spec.DisplayName)
		assert.Equal(t, "Updated Description", resp.Group.Spec.Description)
	})

	t.Run("successful role and metadata update", func(t *testing.T) {
		err := authService.CreateGroup(orgID, models.DomainTypeOrganization, "full-update-group", models.RoleOrgViewer, "Full Update Group", "Full Update Description")
		require.NoError(t, err)

		groupSpec := &pb.Group_Spec{
			Role:        models.RoleOrgAdmin,
			DisplayName: "Full Update Display",
			Description: "Full Update Description",
		}

		resp, err := UpdateGroup(ctx, models.DomainTypeOrganization, orgID, "full-update-group", groupSpec, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, models.RoleOrgAdmin, resp.Group.Spec.Role)
		assert.Equal(t, "Full Update Display", resp.Group.Spec.DisplayName)
		assert.Equal(t, "Full Update Description", resp.Group.Spec.Description)
	})

	t.Run("update preserves group membership", func(t *testing.T) {
		err := authService.CreateGroup(orgID, models.DomainTypeOrganization, "membership-group", models.RoleOrgViewer, "Membership Group", "Membership Description")
		require.NoError(t, err)

		userID1 := uuid.New().String()
		userID2 := uuid.New().String()
		err = authService.AddUserToGroup(orgID, models.DomainTypeOrganization, userID1, "membership-group")
		require.NoError(t, err)
		err = authService.AddUserToGroup(orgID, models.DomainTypeOrganization, userID2, "membership-group")
		require.NoError(t, err)

		groupSpec := &pb.Group_Spec{
			Role: models.RoleOrgAdmin,
		}

		resp, err := UpdateGroup(ctx, models.DomainTypeOrganization, orgID, "membership-group", groupSpec, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		users, err := authService.GetGroupUsers(orgID, models.DomainTypeOrganization, "membership-group")
		require.NoError(t, err)
		assert.Contains(t, users, userID1)
		assert.Contains(t, users, userID2)
		assert.Len(t, users, 2)
	})

	t.Run("canvas group update", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		err = authService.CreateGroup(canvasID, models.DomainTypeCanvas, "canvas-group", models.RoleCanvasViewer, "Canvas Group", "Canvas Description")
		require.NoError(t, err)

		groupSpec := &pb.Group_Spec{
			Role: models.RoleCanvasAdmin,
		}

		resp, err := UpdateGroup(ctx, models.DomainTypeCanvas, canvasID, "canvas-group", groupSpec, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, models.RoleCanvasAdmin, resp.Group.Spec.Role)
	})

	t.Run("group not found", func(t *testing.T) {
		groupSpec := &pb.Group_Spec{
			Role: models.RoleOrgAdmin,
		}

		_, err := UpdateGroup(ctx, models.DomainTypeOrganization, orgID, "non-existent-group", groupSpec, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("missing group name", func(t *testing.T) {
		groupSpec := &pb.Group_Spec{
			Role: models.RoleOrgAdmin,
		}

		_, err := UpdateGroup(ctx, models.DomainTypeOrganization, orgID, "", groupSpec, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})
}
