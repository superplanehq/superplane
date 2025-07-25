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

func Test_RemoveUserFromGroup(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create a group first
	err = authService.CreateGroup(orgID, models.DomainTypeOrg, "test-group", models.RoleOrgAdmin, "Test Group", "Test group description")
	require.NoError(t, err)

	// Add user to group first
	err = authService.AddUserToGroup(orgID, models.DomainTypeOrg, r.User.String(), "test-group")
	require.NoError(t, err)

	t.Run("successful remove user from group with user ID", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrg, orgID, r.User.String(), "", "test-group", authService)
		require.NoError(t, err)
	})

	t.Run("successful remove user from group with user email", func(t *testing.T) {
		testEmail := "test-remove-group@example.com"

		user := &models.User{
			Name:     testEmail,
			IsActive: false,
		}
		err = user.Create()
		require.NoError(t, err)

		accountProvider := &models.AccountProvider{
			Provider: "github",
			UserID:   user.ID,
			Email:    testEmail,
		}
		err = accountProvider.Create()
		require.NoError(t, err)

		err = authService.CreateGroup(orgID, "org", "test-group-email-remove", models.RoleOrgAdmin, "Test Group Email Remove", "Test group email remove description")
		require.NoError(t, err)

		err = authService.AddUserToGroup(orgID, "org", user.ID.String(), "test-group-email-remove")
		require.NoError(t, err)

		_, err = RemoveUserFromGroup(ctx, models.DomainTypeOrg, orgID, "", testEmail, "test-group-email-remove", authService)
		require.NoError(t, err)
	})

	t.Run("user not found by email", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrg, orgID, "", "nonexistent@example.com", "test-group", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrg, orgID, r.User.String(), "", "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrg, orgID, "", "", "test-group", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrg, orgID, "invalid-uuid", "", "test-group", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("successful canvas group remove user", func(t *testing.T) {
		canvasID := uuid.New().String()

		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)
		err = authService.CreateGroup(canvasID, models.DomainTypeCanvas, "canvas-group", models.RoleCanvasAdmin, "Canvas Group", "Canvas group description")
		require.NoError(t, err)
		err = authService.AddUserToGroup(canvasID, models.DomainTypeCanvas, r.User.String(), "canvas-group")
		require.NoError(t, err)

		_, err = RemoveUserFromGroup(ctx, models.DomainTypeCanvas, canvasID, r.User.String(), "", "canvas-group", authService)
		require.NoError(t, err)
	})
}
