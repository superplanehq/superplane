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

func Test_AssignRole(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful role assignment with user ID", func(t *testing.T) {
		resp, err := AssignRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, r.User.String(), "", authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("successful role assignment with user email", func(t *testing.T) {
		testEmail := "test@example.com"
		resp, err := AssignRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "", testEmail, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify user was created
		user, err := models.FindInactiveUserByEmail(testEmail)
		require.NoError(t, err)
		assert.Equal(t, testEmail, user.Name)
		assert.False(t, user.IsActive)
	})

	t.Run("invalid request - missing role", func(t *testing.T) {
		_, err := AssignRole(ctx, models.DomainTypeOrganization, orgID, "", r.User.String(), "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := AssignRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "", "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := AssignRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "invalid-uuid", "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})
}
