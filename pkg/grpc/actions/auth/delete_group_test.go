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

func Test_DeleteOrganizationGroup(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful group deletion", func(t *testing.T) {

		err := authService.CreateGroup(orgID, models.DomainTypeOrg, "test-group", models.RoleOrgAdmin)
		require.NoError(t, err)

		userID := uuid.New().String()
		err = authService.AddUserToGroup(orgID, models.DomainTypeOrg, userID, "test-group")
		require.NoError(t, err)

		groups, err := authService.GetGroups(orgID, models.DomainTypeOrg)
		require.NoError(t, err)
		assert.Contains(t, groups, "test-group")

		users, err := authService.GetGroupUsers(orgID, models.DomainTypeOrg, "test-group")
		require.NoError(t, err)
		assert.Contains(t, users, userID)

		req := &pb.DeleteGroupRequest{
			DomainId:   orgID,
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			GroupName:  "test-group",
		}

		resp, err := DeleteGroup(ctx, "org", orgID, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		users, err = authService.GetGroupUsers(orgID, models.DomainTypeOrg, "test-group")
		require.NoError(t, err)
		assert.Empty(t, users)

		// Verify the group no longer exists in the groups list
		groups, err = authService.GetGroups(orgID, models.DomainTypeOrg)
		require.NoError(t, err)
		assert.NotContains(t, groups, "test-group")
	})

	t.Run("delete non-existent group", func(t *testing.T) {
		req := &pb.DeleteGroupRequest{
			DomainId:   orgID,
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			GroupName:  "non-existent-group",
		}

		_, err := DeleteGroup(ctx, "org", orgID, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &pb.DeleteGroupRequest{
			DomainId:   orgID,
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			GroupName:  "",
		}

		_, err := DeleteGroup(ctx, "org", orgID, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - invalid organization ID for group", func(t *testing.T) {
		err := authService.CreateGroup(orgID, models.DomainTypeOrg, "test-group", models.RoleOrgAdmin)
		require.NoError(t, err)

		req := &pb.DeleteGroupRequest{
			DomainId:   "invalid-uuid",
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			GroupName:  "test-group",
		}

		_, err = DeleteGroup(ctx, "org", "invalid-uuid", req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})
}

func Test_DeleteCanvasGroup(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	canvasID := uuid.New().String()
	err := authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	t.Run("successful canvas group deletion", func(t *testing.T) {

		err := authService.CreateGroup(canvasID, models.DomainTypeCanvas, "canvas-group", models.RoleCanvasAdmin)
		require.NoError(t, err)

		userID := uuid.New().String()
		err = authService.AddUserToGroup(canvasID, models.DomainTypeCanvas, userID, "canvas-group")
		require.NoError(t, err)

		groups, err := authService.GetGroups(canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)
		assert.Contains(t, groups, "canvas-group")

		users, err := authService.GetGroupUsers(canvasID, models.DomainTypeCanvas, "canvas-group")
		require.NoError(t, err)
		assert.Contains(t, users, userID)

		req := &pb.DeleteGroupRequest{
			DomainId:   canvasID,
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_CANVAS,
			GroupName:  "canvas-group",
		}

		resp, err := DeleteGroup(ctx, "canvas", canvasID, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		users, err = authService.GetGroupUsers(canvasID, models.DomainTypeCanvas, "canvas-group")
		require.NoError(t, err)
		assert.Empty(t, users)

		// Verify the group no longer exists in the groups list
		groups, err = authService.GetGroups(canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)
		assert.NotContains(t, groups, "canvas-group")
	})

	t.Run("delete non-existent canvas group", func(t *testing.T) {
		req := &pb.DeleteGroupRequest{
			DomainId:   canvasID,
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_CANVAS,
			GroupName:  "non-existent-group",
		}

		_, err := DeleteGroup(ctx, "canvas", canvasID, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &pb.DeleteGroupRequest{
			DomainId:   canvasID,
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_CANVAS,
			GroupName:  "",
		}

		_, err := DeleteGroup(ctx, "canvas", canvasID, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - invalid canvas ID", func(t *testing.T) {
		req := &pb.DeleteGroupRequest{
			DomainId:   "invalid-uuid",
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_CANVAS,
			GroupName:  "test-group",
		}

		_, err := DeleteGroup(ctx, "canvas", "invalid-uuid", req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})
}
