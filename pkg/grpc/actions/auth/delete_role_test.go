package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/metadata"
)

func Test_DeleteRole(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

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

	err := r.AuthService.CreateCustomRole(orgID, customRoleDef)
	require.NoError(t, err)

	t.Run("successful custom role deletion", func(t *testing.T) {
		roleDef, err := r.AuthService.GetRoleDefinition("test-custom-role-to-delete", models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, "test-custom-role-to-delete", roleDef.Name)

		resp, err := DeleteRole(ctx, models.DomainTypeOrganization, orgID, "test-custom-role-to-delete", r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		_, err = r.AuthService.GetRoleDefinition("test-custom-role-to-delete", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)
	})

	t.Run("invalid request - missing role name", func(t *testing.T) {
		_, err := DeleteRole(ctx, models.DomainTypeOrganization, orgID, "", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role name must be specified")
	})

	t.Run("invalid request - invalid domain type", func(t *testing.T) {
		_, err := DeleteRole(ctx, "invalid-domain-type", orgID, "test-role", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("invalid request - default role name", func(t *testing.T) {
		_, err := DeleteRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete default role")
	})

	t.Run("invalid request - nonexistent role", func(t *testing.T) {
		_, err := DeleteRole(ctx, models.DomainTypeOrganization, orgID, "nonexistent-role", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("invalid request - invalid UUID", func(t *testing.T) {
		_, err := DeleteRole(ctx, models.DomainTypeOrganization, "invalid-uuid", "test-role", r.AuthService)
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
		err = r.AuthService.CreateCustomRole(orgID, customRoleWithUsers)
		require.NoError(t, err)

		userID := uuid.New().String()
		err = r.AuthService.AssignRole(userID, "test-role-with-users", orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		resp, err := DeleteRole(ctx, models.DomainTypeOrganization, orgID, "test-role-with-users", r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		_, err = r.AuthService.GetRoleDefinition("test-role-with-users", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)

		userRoles, err := r.AuthService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)
		for _, role := range userRoles {
			assert.NotEqual(t, "test-role-with-users", role.Name)
		}
	})

	t.Run("delete global canvas role with organization context", func(t *testing.T) {
		err := r.AuthService.SetupGlobalCanvasRoles(orgID)
		require.NoError(t, err)

		md := metadata.Pairs("x-organization-id", orgID)
		ctxWithOrg := metadata.NewIncomingContext(ctx, md)

		globalRoleDef := &authorization.RoleDefinition{
			Name:        "global-canvas-deletable-role",
			DisplayName: "Global Canvas Deletable Role",
			Description: "Role to be deleted",
			DomainType:  models.DomainTypeCanvas,
			Permissions: []*authorization.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: models.DomainTypeCanvas,
				},
			},
		}
		err = r.AuthService.CreateCustomRoleWithOrgContext("*", orgID, globalRoleDef)
		require.NoError(t, err)

		_, err = r.AuthService.GetGlobalCanvasRoleDefinition("global-canvas-deletable-role", orgID)
		require.NoError(t, err)

		resp, err := DeleteRole(ctxWithOrg, models.DomainTypeCanvas, "*", "global-canvas-deletable-role", r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		_, err = r.AuthService.GetGlobalCanvasRoleDefinition("global-canvas-deletable-role", orgID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("delete global canvas role without organization context fails", func(t *testing.T) {
		// Create another global canvas role first
		globalRoleDef2 := &authorization.RoleDefinition{
			Name:        "global-canvas-deletable-role-2",
			DisplayName: "Global Canvas Deletable Role 2",
			Description: "Another role to test deletion failure",
			DomainType:  models.DomainTypeCanvas,
			Permissions: []*authorization.Permission{
				{
					Resource:   "canvas",
					Action:     "write",
					DomainType: models.DomainTypeCanvas,
				},
			},
		}
		err = r.AuthService.CreateCustomRoleWithOrgContext("*", orgID, globalRoleDef2)
		require.NoError(t, err)

		_, err := DeleteRole(ctx, models.DomainTypeCanvas, "*", "global-canvas-deletable-role-2", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "organization context required for global canvas roles")

		_, err = r.AuthService.GetGlobalCanvasRoleDefinition("global-canvas-deletable-role-2", orgID)
		require.NoError(t, err)
	})

	t.Run("cannot delete default global canvas roles", func(t *testing.T) {
		err := r.AuthService.SetupGlobalCanvasRoles(orgID)
		require.NoError(t, err)

		md := metadata.Pairs("x-organization-id", orgID)
		ctxWithOrg := metadata.NewIncomingContext(ctx, md)

		_, deleteErr := DeleteRole(ctxWithOrg, models.DomainTypeCanvas, "*", models.RoleCanvasViewer, r.AuthService)
		assert.Error(t, deleteErr)
		assert.Contains(t, deleteErr.Error(), "cannot delete default role")
	})
}
