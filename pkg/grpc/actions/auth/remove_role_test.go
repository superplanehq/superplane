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

func Test_RemoveRole(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Assign role first
	err = authService.AssignRole(r.User.String(), models.RoleOrgAdmin, orgID, models.DomainTypeOrg)
	require.NoError(t, err)

	t.Run("successful role removal with user ID", func(t *testing.T) {
		resp, err := RemoveRole(ctx, models.DomainTypeOrg, orgID, models.RoleOrgAdmin, r.User.String(), "", authService)
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

		err = authService.AssignRole(user.ID.String(), models.RoleOrgAdmin, orgID, models.DomainTypeOrg)
		require.NoError(t, err)

		resp, err := RemoveRole(ctx, models.DomainTypeOrg, orgID, models.RoleOrgAdmin, "", testEmail, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("user not found by email", func(t *testing.T) {
		_, err := RemoveRole(ctx, models.DomainTypeOrg, orgID, models.RoleOrgAdmin, "", "nonexistent@example.com", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := RemoveRole(ctx, models.DomainTypeOrg, orgID, models.RoleOrgAdmin, "", "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := RemoveRole(ctx, models.DomainTypeOrg, orgID, models.RoleOrgAdmin, "invalid-uuid", "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})
}
