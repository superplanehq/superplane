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

func Test_AssignRole(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)

	err := authService.SetupOrganizationRoles(r.Organization.ID.String())
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())

	t.Run("assign role with ID", func(t *testing.T) {
		resp, err := AssignRole(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), models.RoleOrgAdmin, r.User.String(), "", authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("assign role with email", func(t *testing.T) {
		email := "test@example.com"
		resp, err := AssignRole(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), models.RoleOrgAdmin, "", email, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		//
		// Verify user was created
		// TODO: this should create an invitation and not an inactive user?
		//
		user, err := models.FindInactiveUserByEmail(email, r.Organization.ID)
		require.NoError(t, err)
		assert.Equal(t, email, user.Name)
		assert.False(t, user.IsActive)
	})

	t.Run("invalid request - missing role", func(t *testing.T) {
		_, err := AssignRole(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), "", r.User.String(), "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := AssignRole(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), models.RoleOrgAdmin, "", "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := AssignRole(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), models.RoleOrgAdmin, "invalid-uuid", "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})
}
