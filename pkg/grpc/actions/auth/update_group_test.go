package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
)

func TestUpdateGroup(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful role update", func(t *testing.T) {
		// Create a group first
		err := CreateGroupWithMetadata(orgID, models.DomainTypeOrg, "test-group", models.RoleOrgViewer, "Test Group", "Test Description", authService)
		require.NoError(t, err)

		// Update the group role
		req := &pb.UpdateGroupRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			GroupName:  "test-group",
			Role:       models.RoleOrgAdmin,
		}

		resp, err := UpdateGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, models.RoleOrgAdmin, resp.Group.Spec.Role)

		// Verify the role was updated
		role, err := authService.GetGroupRole(orgID, models.DomainTypeOrg, "test-group")
		require.NoError(t, err)
		assert.Equal(t, models.RoleOrgAdmin, role)
	})

	t.Run("successful metadata update", func(t *testing.T) {
		// Create a group first
		err := CreateGroupWithMetadata(orgID, models.DomainTypeOrg, "metadata-group", models.RoleOrgViewer, "Metadata Group", "Metadata Description", authService)
		require.NoError(t, err)

		// Update the group metadata
		req := &pb.UpdateGroupRequest{
			DomainType:  pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:    orgID,
			GroupName:   "metadata-group",
			DisplayName: "Updated Display Name",
			Description: "Updated Description",
		}

		resp, err := UpdateGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "Updated Display Name", resp.Group.Spec.DisplayName)
		assert.Equal(t, "Updated Description", resp.Group.Spec.Description)
	})

	t.Run("successful role and metadata update", func(t *testing.T) {
		// Create a group first
		err := CreateGroupWithMetadata(orgID, models.DomainTypeOrg, "full-update-group", models.RoleOrgViewer, "Full Update Group", "Full Update Description", authService)
		require.NoError(t, err)

		// Update both role and metadata
		req := &pb.UpdateGroupRequest{
			DomainType:  pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:    orgID,
			GroupName:   "full-update-group",
			Role:        models.RoleOrgAdmin,
			DisplayName: "Full Update Display",
			Description: "Full Update Description",
		}

		resp, err := UpdateGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, models.RoleOrgAdmin, resp.Group.Spec.Role)
		assert.Equal(t, "Full Update Display", resp.Group.Spec.DisplayName)
		assert.Equal(t, "Full Update Description", resp.Group.Spec.Description)
	})

	t.Run("update preserves group membership", func(t *testing.T) {
		// Create a group first
		err := CreateGroupWithMetadata(orgID, models.DomainTypeOrg, "membership-group", models.RoleOrgViewer, "Membership Group", "Membership Description", authService)
		require.NoError(t, err)

		// Add users to the group
		userID1 := uuid.New().String()
		userID2 := uuid.New().String()
		err = authService.AddUserToGroup(orgID, models.DomainTypeOrg, userID1, "membership-group")
		require.NoError(t, err)
		err = authService.AddUserToGroup(orgID, models.DomainTypeOrg, userID2, "membership-group")
		require.NoError(t, err)

		// Update the group role
		req := &pb.UpdateGroupRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			GroupName:  "membership-group",
			Role:       models.RoleOrgAdmin,
		}

		resp, err := UpdateGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify users are still in the group
		users, err := authService.GetGroupUsers(orgID, models.DomainTypeOrg, "membership-group")
		require.NoError(t, err)
		assert.Contains(t, users, userID1)
		assert.Contains(t, users, userID2)
		assert.Len(t, users, 2)
	})

	t.Run("canvas group update", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		// Create a canvas group first
		err = CreateGroupWithMetadata(canvasID, models.DomainTypeCanvas, "canvas-group", models.RoleCanvasViewer, "Canvas Group", "Canvas Description", authService)
		require.NoError(t, err)

		// Update the canvas group
		req := &pb.UpdateGroupRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_CANVAS,
			DomainId:   canvasID,
			GroupName:  "canvas-group",
			Role:       models.RoleCanvasAdmin,
		}

		resp, err := UpdateGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, models.RoleCanvasAdmin, resp.Group.Spec.Role)
	})

	t.Run("group not found", func(t *testing.T) {
		req := &pb.UpdateGroupRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			GroupName:  "non-existent-group",
			Role:       models.RoleOrgAdmin,
		}

		_, err := UpdateGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("invalid domain ID", func(t *testing.T) {
		req := &pb.UpdateGroupRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   "invalid-uuid",
			GroupName:  "test-group",
			Role:       models.RoleOrgAdmin,
		}

		_, err := UpdateGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid domain ID")
	})

	t.Run("missing group name", func(t *testing.T) {
		req := &pb.UpdateGroupRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			GroupName:  "",
			Role:       models.RoleOrgAdmin,
		}

		_, err := UpdateGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("no fields to update", func(t *testing.T) {
		req := &pb.UpdateGroupRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			GroupName:  "test-group",
			// No fields to update
		}

		_, err := UpdateGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one field must be provided for update")
	})

	t.Run("invalid domain type", func(t *testing.T) {
		req := &pb.UpdateGroupRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainId:   orgID,
			GroupName:  "test-group",
			Role:       models.RoleOrgAdmin,
		}

		_, err := UpdateGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})
}
