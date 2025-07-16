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

func Test_DeleteOrganizationGroup(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful group deletion", func(t *testing.T) {

		err := authService.CreateGroup(orgID, authorization.DomainOrg, "test-group", authorization.RoleOrgAdmin)
		require.NoError(t, err)

		userID := uuid.New().String()
		err = authService.AddUserToGroup(orgID, authorization.DomainOrg, userID, "test-group")
		require.NoError(t, err)

		groups, err := authService.GetGroups(orgID, authorization.DomainOrg)
		require.NoError(t, err)
		assert.Contains(t, groups, "test-group")

		users, err := authService.GetGroupUsers(orgID, authorization.DomainOrg, "test-group")
		require.NoError(t, err)
		assert.Contains(t, users, userID)

		req := &pb.DeleteOrganizationGroupRequest{
			OrganizationId: orgID,
			GroupName:      "test-group",
		}

		resp, err := DeleteOrganizationGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		users, err = authService.GetGroupUsers(orgID, authorization.DomainOrg, "test-group")
		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("delete non-existent group", func(t *testing.T) {
		req := &pb.DeleteOrganizationGroupRequest{
			OrganizationId: orgID,
			GroupName:      "non-existent-group",
		}

		_, err := DeleteOrganizationGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &pb.DeleteOrganizationGroupRequest{
			OrganizationId: orgID,
			GroupName:      "",
		}

		_, err := DeleteOrganizationGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - invalid organization ID", func(t *testing.T) {
		req := &pb.DeleteOrganizationGroupRequest{
			OrganizationId: "invalid-uuid",
			GroupName:      "test-group",
		}

		_, err := DeleteOrganizationGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UUIDs")
	})
}

func Test_DeleteCanvasGroup(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	canvasID := uuid.New().String()
	err := authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	t.Run("successful canvas group deletion", func(t *testing.T) {

		err := authService.CreateGroup(canvasID, authorization.DomainCanvas, "canvas-group", authorization.RoleCanvasAdmin)
		require.NoError(t, err)

		userID := uuid.New().String()
		err = authService.AddUserToGroup(canvasID, authorization.DomainCanvas, userID, "canvas-group")
		require.NoError(t, err)

		groups, err := authService.GetGroups(canvasID, authorization.DomainCanvas)
		require.NoError(t, err)
		assert.Contains(t, groups, "canvas-group")

		users, err := authService.GetGroupUsers(canvasID, authorization.DomainCanvas, "canvas-group")
		require.NoError(t, err)
		assert.Contains(t, users, userID)

		req := &pb.DeleteCanvasGroupRequest{
			CanvasIdOrName: canvasID,
			GroupName:      "canvas-group",
		}

		resp, err := DeleteCanvasGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		users, err = authService.GetGroupUsers(canvasID, authorization.DomainCanvas, "canvas-group")
		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("delete non-existent canvas group", func(t *testing.T) {
		req := &pb.DeleteCanvasGroupRequest{
			CanvasIdOrName: canvasID,
			GroupName:      "non-existent-group",
		}

		_, err := DeleteCanvasGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &pb.DeleteCanvasGroupRequest{
			CanvasIdOrName: canvasID,
			GroupName:      "",
		}

		_, err := DeleteCanvasGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - invalid canvas ID", func(t *testing.T) {
		req := &pb.DeleteCanvasGroupRequest{
			CanvasIdOrName: "invalid-uuid",
			GroupName:      "test-group",
		}

		_, err := DeleteCanvasGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "canvas not found")
	})
}
