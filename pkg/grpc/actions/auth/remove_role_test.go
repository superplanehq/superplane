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

func Test_RemoveRole(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	err := authService.SetupOrganizationRoles(r.Organization.ID.String())
	require.NoError(t, err)

	// Assign role first
	err = authService.AssignRole(r.User.String(), models.RoleOrgAdmin, r.Organization.ID.String(), models.DomainTypeOrganization)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())

	t.Run("remove role with user ID", func(t *testing.T) {
		resp, err := RemoveRole(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), models.RoleOrgAdmin, r.User.String(), "", authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("remove role with user email", func(t *testing.T) {
		email := "test-remove@example.com"

		user := &models.User{
			Name:           email,
			IsActive:       false,
			OrganizationID: r.Organization.ID,
		}

		err := user.Create()
		require.NoError(t, err)

		accountProvider := &models.AccountProvider{
			Provider: "github",
			UserID:   user.ID,
			Email:    email,
		}

		err = accountProvider.Create()
		require.NoError(t, err)

		err = authService.AssignRole(user.ID.String(), models.RoleOrgAdmin, r.Organization.ID.String(), models.DomainTypeOrganization)
		require.NoError(t, err)

		resp, err := RemoveRole(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), models.RoleOrgAdmin, "", email, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("user not found by email", func(t *testing.T) {
		_, err := RemoveRole(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), models.RoleOrgAdmin, "", "nonexistent@example.com", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := RemoveRole(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), models.RoleOrgAdmin, "", "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := RemoveRole(ctx, models.DomainTypeOrganization, r.Organization.ID.String(), models.RoleOrgAdmin, "invalid-uuid", "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})
}
