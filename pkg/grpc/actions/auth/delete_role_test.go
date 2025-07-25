package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test_DeleteRole(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	customRoleDef := &authorization.RoleDefinition{
		Name:       "test-custom-role-to-delete",
		DomainType: models.DomainTypeOrganization,
		Permissions: []*authorization.Permission{
			{
				Resource:   "canvas",
				Action:     "read",
				DomainType: models.DomainTypeOrganization,
			},
			{
				Resource:   "canvas",
				Action:     "write",
				DomainType: models.DomainTypeOrganization,
			},
		},
	}
	err = authService.CreateCustomRole(orgID, customRoleDef)
	require.NoError(t, err)

	t.Run("successful custom role deletion", func(t *testing.T) {
		roleDef, err := authService.GetRoleDefinition("test-custom-role-to-delete", models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, "test-custom-role-to-delete", roleDef.Name)

		resp, err := DeleteRole(ctx, models.DomainTypeOrganization, orgID, "test-custom-role-to-delete", authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		_, err = authService.GetRoleDefinition("test-custom-role-to-delete", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)
	})

	t.Run("invalid request - missing role name", func(t *testing.T) {
		_, err := DeleteRole(ctx, models.DomainTypeOrganization, orgID, "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role name must be specified")
	})

	t.Run("invalid request - invalid domain type", func(t *testing.T) {
		_, err := DeleteRole(ctx, "invalid-domain-type", orgID, "test-role", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("invalid request - default role name", func(t *testing.T) {
		_, err := DeleteRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete default role")
	})

	t.Run("invalid request - nonexistent role", func(t *testing.T) {
		_, err := DeleteRole(ctx, models.DomainTypeOrganization, orgID, "nonexistent-role", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("invalid request - invalid UUID", func(t *testing.T) {
		_, err := DeleteRole(ctx, models.DomainTypeOrganization, "invalid-uuid", "test-role", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("delete role that users are assigned to", func(t *testing.T) {
		customRoleWithUsers := &authorization.RoleDefinition{
			Name:       "test-role-with-users",
			DomainType: models.DomainTypeOrganization,
			Permissions: []*authorization.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: models.DomainTypeOrganization,
				},
			},
		}
		err = authService.CreateCustomRole(orgID, customRoleWithUsers)
		require.NoError(t, err)

		userID := uuid.New().String()
		err = authService.AssignRole(userID, "test-role-with-users", orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		resp, err := DeleteRole(ctx, models.DomainTypeOrganization, orgID, "test-role-with-users", authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		_, err = authService.GetRoleDefinition("test-role-with-users", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)

		userRoles, err := authService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)
		for _, role := range userRoles {
			assert.NotEqual(t, "test-role-with-users", role.Name)
		}
	})
}
