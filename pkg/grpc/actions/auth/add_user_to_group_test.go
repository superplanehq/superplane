package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test_AddUserToGroup(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create a group first
	err = authService.CreateGroup(orgID, models.DomainTypeOrg, "test-group", models.RoleOrgAdmin, "Test Group", "Test group description")
	require.NoError(t, err)

	t.Run("successful add user to group with user ID", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, models.DomainTypeOrg, orgID, r.User.String(), "", "test-group", authService)
		require.NoError(t, err)
	})

	t.Run("successful add user to group with user email", func(t *testing.T) {
		testEmail := "test-add-group@example.com"

		err = authService.CreateGroup(orgID, "org", "test-group-email", models.RoleOrgAdmin, "Test Group Email", "Test group email description")
		require.NoError(t, err)

		_, err := AddUserToGroup(ctx, models.DomainTypeOrg, orgID, "", testEmail, "test-group-email", authService)
		require.NoError(t, err)

		user, err := models.FindInactiveUserByEmail(testEmail)
		require.NoError(t, err)
		assert.Equal(t, testEmail, user.Name)
		assert.False(t, user.IsActive)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, models.DomainTypeOrg, orgID, r.User.String(), "", "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, models.DomainTypeOrg, orgID, "", "", "test-group", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user identifier must be specified")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, models.DomainTypeOrg, orgID, "invalid-uuid", "", "test-group", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("canvas groups - group does not exist", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		_, err = AddUserToGroup(ctx, models.DomainTypeCanvas, canvasID, r.User.String(), "", "non-existent-group", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group non-existent-group does not exist")
	})

	t.Run("successful add user to canvas group", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		// Create a canvas group first
		err = authService.CreateGroup(canvasID, models.DomainTypeCanvas, "canvas-test-group", models.RoleCanvasAdmin, "Canvas Test Group", "Canvas test group description")
		require.NoError(t, err)

		_, err = AddUserToGroup(ctx, models.DomainTypeCanvas, canvasID, r.User.String(), "", "canvas-test-group", authService)
		require.NoError(t, err)
	})
}
