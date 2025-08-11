package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test_RemoveUserFromGroup(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	// Create a group first
	require.NoError(t, r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group", models.RoleOrgAdmin, "Test Group", "Test group description"))
	require.NoError(t, r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, r.User.String(), "test-group"))

	t.Run("successful remove user from group with user ID", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrganization, orgID, r.User.String(), "", "test-group", r.AuthService)
		require.NoError(t, err)
	})

	t.Run("remove user from group with email", func(t *testing.T) {
		user := &models.User{
			Name:     "test-remove-group@example.com",
			IsActive: false,
		}

		err := user.Create()
		require.NoError(t, err)

		accountProvider := &models.AccountProvider{Provider: "github", UserID: user.ID, Email: user.Email}
		err = accountProvider.Create()
		require.NoError(t, err)

		err = r.AuthService.CreateGroup(orgID, "org", "test-group-email-remove", models.RoleOrgAdmin, "Test Group Email Remove", "Test group email remove description")
		require.NoError(t, err)

		err = r.AuthService.AddUserToGroup(orgID, "org", user.ID.String(), "test-group-email-remove")
		require.NoError(t, err)

		_, err = RemoveUserFromGroup(ctx, models.DomainTypeOrganization, orgID, "", user.Email, "test-group-email-remove", r.AuthService)
		require.NoError(t, err)
	})

	t.Run("user not found by email", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrganization, orgID, "", "nonexistent@example.com", "test-group", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrganization, orgID, r.User.String(), "", "", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrganization, orgID, "", "", "test-group", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrganization, orgID, "invalid-uuid", "", "test-group", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("successful canvas group remove user", func(t *testing.T) {
		require.NoError(t, r.AuthService.CreateGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, "canvas-group", models.RoleCanvasAdmin, "Canvas Group", "Canvas group description"))
		require.NoError(t, r.AuthService.AddUserToGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, r.User.String(), "canvas-group"))

		_, err := RemoveUserFromGroup(ctx, models.DomainTypeCanvas, r.Canvas.ID.String(), r.User.String(), "", "canvas-group", r.AuthService)
		require.NoError(t, err)
	})
}
