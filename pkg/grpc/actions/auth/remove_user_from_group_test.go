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
	err := r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group", models.RoleOrgAdmin, "Test Group", "Test group description")
	require.NoError(t, err)

	// Add user to group first
	err = r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, r.User.String(), "test-group")
	require.NoError(t, err)

	t.Run("successful remove user from group with user ID", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, orgID, models.DomainTypeOrganization, orgID, r.User.String(), "", "test-group", r.AuthService)
		require.NoError(t, err)
	})

	t.Run("user not found by email", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, orgID, models.DomainTypeOrganization, orgID, "", "nonexistent@example.com", "test-group", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, orgID, models.DomainTypeOrganization, orgID, r.User.String(), "", "", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, orgID, models.DomainTypeOrganization, orgID, "", "", "test-group", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, orgID, models.DomainTypeOrganization, orgID, "invalid-uuid", "", "test-group", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("successful canvas group remove user", func(t *testing.T) {
		err = r.AuthService.CreateGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, "canvas-group", models.RoleCanvasAdmin, "Canvas Group", "Canvas group description")
		require.NoError(t, err)
		err = r.AuthService.AddUserToGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, r.User.String(), "canvas-group")
		require.NoError(t, err)

		_, err = RemoveUserFromGroup(ctx, orgID, models.DomainTypeCanvas, r.Canvas.ID.String(), r.User.String(), "", "canvas-group", r.AuthService)
		require.NoError(t, err)
	})
}
