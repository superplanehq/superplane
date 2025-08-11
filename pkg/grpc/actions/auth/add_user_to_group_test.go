package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test_AddUserToGroup(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	// Create a group first
	err := r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group", models.RoleOrgAdmin, "Test Group", "Test group description")
	require.NoError(t, err)

	t.Run("successful add user to group with user ID", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, models.DomainTypeOrganization, orgID, r.User.String(), "", "test-group", r.AuthService)
		require.NoError(t, err)
	})

	t.Run("successful add user to group with user email", func(t *testing.T) {
		testEmail := "test-add-group@example.com"

		err = r.AuthService.CreateGroup(orgID, "org", "test-group-email", models.RoleOrgAdmin, "Test Group Email", "Test group email description")
		require.NoError(t, err)

		_, err := AddUserToGroup(ctx, models.DomainTypeOrganization, orgID, "", testEmail, "test-group-email", r.AuthService)
		require.NoError(t, err)

		user, err := models.FindInactiveUserByEmail(testEmail)
		require.NoError(t, err)
		assert.Equal(t, testEmail, user.Name)
		assert.False(t, user.IsActive)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, models.DomainTypeOrganization, orgID, r.User.String(), "", "", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, models.DomainTypeOrganization, orgID, "", "", "test-group", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user identifier must be specified")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, models.DomainTypeOrganization, orgID, "invalid-uuid", "", "test-group", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("canvas groups - group does not exist", func(t *testing.T) {
		_, err = AddUserToGroup(ctx, models.DomainTypeCanvas, r.Canvas.ID.String(), r.User.String(), "", "non-existent-group", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group non-existent-group does not exist")
	})

	t.Run("successful add user to canvas group", func(t *testing.T) {
		// Create a canvas group first
		err = r.AuthService.CreateGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, "canvas-test-group", models.RoleCanvasAdmin, "Canvas Test Group", "Canvas test group description")
		require.NoError(t, err)

		_, err = AddUserToGroup(ctx, models.DomainTypeCanvas, r.Canvas.ID.String(), r.User.String(), "", "canvas-test-group", r.AuthService)
		require.NoError(t, err)
	})
}
