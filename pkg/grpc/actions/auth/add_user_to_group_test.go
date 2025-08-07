package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test_AddUserToGroup(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)

	err := authService.SetupOrganizationRoles(r.Organization.ID.String())
	require.NoError(t, err)
	err = authService.SetupCanvasRoles(r.Canvas.ID.String())
	require.NoError(t, err)

	// Create a group first
	groupName := support.RandomName("group")
	err = authService.CreateGroup(r.Organization.ID.String(), models.DomainTypeOrganization, groupName, models.RoleOrgAdmin, groupName, "")
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())

	t.Run("add user to organization group with ID", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), r.User.String(), "", groupName, authService)
		require.NoError(t, err)

		// TODO: we need to do some assertion here
	})

	t.Run("add user to organization group with email", func(t *testing.T) {
		email := "test-add-group@example.com"

		_, err := AddUserToGroup(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), "", email, groupName, authService)
		require.NoError(t, err)

		user, err := models.FindInactiveUserByEmail(email, r.Organization.ID)
		require.NoError(t, err)
		assert.Equal(t, email, user.Name)
		assert.False(t, user.IsActive)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), r.User.String(), "", "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), "", "", groupName, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user identifier must be specified")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), "invalid-uuid", "", groupName, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("canvas group does not exist - error", func(t *testing.T) {
		_, err = AddUserToGroup(ctx, models.DomainTypeCanvas, r.Canvas.ID.String(), r.User.String(), "", "non-existent-group", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group non-existent-group does not exist")
	})

	t.Run("add user to canvas group", func(t *testing.T) {
		canvasGroupName := support.RandomName("canvas-group")
		err = authService.CreateGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, canvasGroupName, models.RoleCanvasAdmin, canvasGroupName, "")
		require.NoError(t, err)

		_, err = AddUserToGroup(ctx, models.DomainTypeCanvas, r.Canvas.ID.String(), r.User.String(), "", canvasGroupName, authService)
		require.NoError(t, err)

		// TODO: we need to do some assertion here
	})
}
