package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test_RemoveRole(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	// Assign role first
	require.NoError(t, r.AuthService.AssignRole(r.User.String(), models.RoleOrgAdmin, orgID, models.DomainTypeOrganization))

	t.Run("successful role removal with user ID", func(t *testing.T) {
		resp, err := RemoveRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, r.User.String(), "", r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("successful role removal with user email", func(t *testing.T) {
		testEmail := "test-remove@example.com"

		user := &models.User{
			Name:     testEmail,
			IsActive: false,
		}
		err := user.Create()
		require.NoError(t, err)

		accountProvider := &models.AccountProvider{
			Provider: "github",
			UserID:   user.ID,
			Email:    testEmail,
		}
		err = accountProvider.Create()
		require.NoError(t, err)

		err = r.AuthService.AssignRole(user.ID.String(), models.RoleOrgAdmin, orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		resp, err := RemoveRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "", testEmail, r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("user not found by email", func(t *testing.T) {
		_, err := RemoveRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "", "nonexistent@example.com", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := RemoveRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "", "", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := RemoveRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "invalid-uuid", "", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})
}
