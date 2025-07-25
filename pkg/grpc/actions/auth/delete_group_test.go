package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test_DeleteOrganizationGroup(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful group deletion", func(t *testing.T) {

		err := authService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group", models.RoleOrgAdmin, "Test Group", "Test Group")
		require.NoError(t, err)

		userID := uuid.New().String()
		err = authService.AddUserToGroup(orgID, models.DomainTypeOrganization, userID, "test-group")
		require.NoError(t, err)

		groups, err := authService.GetGroups(orgID, models.DomainTypeOrganization)
		require.NoError(t, err)
		assert.Contains(t, groups, "test-group")

		users, err := authService.GetGroupUsers(orgID, models.DomainTypeOrganization, "test-group")
		require.NoError(t, err)
		assert.Contains(t, users, userID)

		resp, err := DeleteGroup(ctx, models.DomainTypeOrganization, orgID, "test-group", authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		users, err = authService.GetGroupUsers(orgID, models.DomainTypeOrganization, "test-group")
		require.NoError(t, err)
		assert.Empty(t, users)

		// Verify the group no longer exists in the groups list
		groups, err = authService.GetGroups(orgID, models.DomainTypeOrganization)
		require.NoError(t, err)
		assert.NotContains(t, groups, "test-group")
	})

	t.Run("delete non-existent group", func(t *testing.T) {
		_, err := DeleteGroup(ctx, models.DomainTypeOrganization, orgID, "non-existent-group", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := DeleteGroup(ctx, models.DomainTypeOrganization, orgID, "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - invalid organization ID for group", func(t *testing.T) {
		err := authService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group", models.RoleOrgAdmin, "Test Group", "Test Group")
		require.NoError(t, err)

		_, err = DeleteGroup(ctx, models.DomainTypeOrganization, "invalid-uuid", "test-group", authService)
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

		err := authService.CreateGroup(canvasID, models.DomainTypeCanvas, "canvas-group", models.RoleCanvasAdmin, "Canvas Group", "Canvas Group")
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

		resp, err := DeleteGroup(ctx, models.DomainTypeCanvas, canvasID, "canvas-group", authService)
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
		_, err := DeleteGroup(ctx, models.DomainTypeCanvas, canvasID, "non-existent-group", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := DeleteGroup(ctx, models.DomainTypeCanvas, canvasID, "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := DeleteGroup(ctx, models.DomainTypeCanvas, canvasID, "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - invalid canvas ID", func(t *testing.T) {
		_, err := DeleteGroup(ctx, models.DomainTypeCanvas, "invalid-uuid", "test-group", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})
}
