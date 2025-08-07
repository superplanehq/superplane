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

func Test_RemoveUserFromGroup(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)

	err := authService.SetupOrganizationRoles(r.Organization.ID.String())
	require.NoError(t, err)

	// Create a group and add user to group first
	groupName := "test-group"
	err = authService.CreateGroup(r.Organization.ID.String(), models.DomainTypeOrganization, groupName, models.RoleOrgAdmin, groupName, "")
	require.NoError(t, err)
	err = authService.AddUserToGroup(r.Organization.ID.String(), models.DomainTypeOrganization, r.User.String(), groupName)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())

	t.Run("remove user from group with ID", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), r.User.String(), "", groupName, authService)
		require.NoError(t, err)
	})

	t.Run("remove user from group with email", func(t *testing.T) {
		email := "test-remove-group@example.com"

		user := &models.User{Name: email, OrganizationID: r.Organization.ID, IsActive: false}
		err = user.Create()
		require.NoError(t, err)

		accountProvider := &models.AccountProvider{Provider: "github", UserID: user.ID, Email: email}
		err = accountProvider.Create()
		require.NoError(t, err)

		err = authService.AddUserToGroup(r.Organization.ID.String(), models.DomainTypeOrganization, user.ID.String(), groupName)
		require.NoError(t, err)

		_, err = RemoveUserFromGroup(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), "", email, groupName, authService)
		require.NoError(t, err)
	})

	t.Run("user not found by email", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), "", "nonexistent@example.com", groupName, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), r.User.String(), "", "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), "", "", groupName, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), "invalid-uuid", "", groupName, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("remove user from canvas group", func(t *testing.T) {
		err := authService.SetupCanvasRoles(r.Canvas.ID.String())
		require.NoError(t, err)

		groupName := "canvas-group"
		err = authService.CreateGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, groupName, models.RoleCanvasAdmin, groupName, "")
		require.NoError(t, err)
		err = authService.AddUserToGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, r.User.String(), groupName)
		require.NoError(t, err)

		_, err = RemoveUserFromGroup(ctx, models.DomainTypeCanvas, r.Canvas.ID.String(), r.User.String(), "", groupName, authService)
		require.NoError(t, err)
	})
}
