package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
)

func Test_DeleteRole(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create a custom role first
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
		// Verify role exists before deletion
		roleDef, err := authService.GetRoleDefinition("test-custom-role-to-delete", models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, "test-custom-role-to-delete", roleDef.Name)

		req := &pb.DeleteRoleRequest{
			RoleName:   "test-custom-role-to-delete",
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
		}

		resp, err := DeleteRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify role was deleted
		_, err = authService.GetRoleDefinition("test-custom-role-to-delete", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)
	})

	t.Run("invalid request - missing role name", func(t *testing.T) {
		req := &pb.DeleteRoleRequest{
			RoleName:   "",
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
		}

		_, err := DeleteRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role name must be specified")
	})

	t.Run("invalid request - invalid domain type", func(t *testing.T) {
		req := &pb.DeleteRoleRequest{
			RoleName:   "test-role",
			DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainId:   orgID,
		}

		_, err := DeleteRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid domain type")
	})

	t.Run("invalid request - default role name", func(t *testing.T) {
		req := &pb.DeleteRoleRequest{
			RoleName:   authorization.RoleOrgAdmin,
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
		}

		_, err := DeleteRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete default role")
	})

	t.Run("invalid request - nonexistent role", func(t *testing.T) {
		req := &pb.DeleteRoleRequest{
			RoleName:   "nonexistent-role",
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
		}

		_, err := DeleteRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("invalid request - invalid UUID", func(t *testing.T) {
		req := &pb.DeleteRoleRequest{
			RoleName:   "test-role",
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   "invalid-uuid",
		}

		_, err := DeleteRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UUIDs")
	})

	t.Run("delete role that users are assigned to", func(t *testing.T) {
		// Create another custom role
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

		// Assign role to a user
		userID := uuid.New().String()
		err = authService.AssignRole(userID, "test-role-with-users", orgID, models.DomainTypeOrganization)
		require.NoError(t, err)

		req := &pb.DeleteRoleRequest{
			RoleName:   "test-role-with-users",
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
		}

		resp, err := DeleteRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify role was deleted and user no longer has the role
		_, err = authService.GetRoleDefinition("test-role-with-users", models.DomainTypeOrganization, orgID)
		assert.Error(t, err)

		// Verify user no longer has the deleted role
		userRoles, err := authService.GetUserRolesForOrg(userID, orgID)
		require.NoError(t, err)
		for _, role := range userRoles {
			assert.NotEqual(t, "test-role-with-users", role.Name)
		}
	})
}
