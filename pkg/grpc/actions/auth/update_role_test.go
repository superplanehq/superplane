package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
)

func Test_UpdateRole(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create a custom role first
	customRoleDef := &authorization.RoleDefinition{
		Name:       "test-custom-role",
		DomainType: models.DomainTypeOrg,
		Permissions: []*authorization.Permission{
			{
				Resource:   "canvas",
				Action:     "read",
				DomainType: models.DomainTypeOrg,
			},
		},
	}
	err = authService.CreateCustomRole(orgID, customRoleDef)
	require.NoError(t, err)

	t.Run("successful custom role update", func(t *testing.T) {
		req := &pb.UpdateRoleRequest{
			RoleName:   "test-custom-role",
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
				{
					Resource:   "canvas",
					Action:     "write",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
				{
					Resource:   "secret",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		resp, err := UpdateRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Check if role was updated by verifying permissions
		roleDef, err := authService.GetRoleDefinition("test-custom-role", models.DomainTypeOrg, orgID)
		require.NoError(t, err)
		assert.Equal(t, "test-custom-role", roleDef.Name)
		assert.Len(t, roleDef.Permissions, 3)
	})

	t.Run("successful custom role update with inheritance", func(t *testing.T) {
		req := &pb.UpdateRoleRequest{
			RoleName:      "test-custom-role",
			DomainType:    pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:      orgID,
			InheritedRole: models.RoleOrgViewer,
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "create",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		resp, err := UpdateRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Check if role was updated with inheritance
		roleDef, err := authService.GetRoleDefinition("test-custom-role", models.DomainTypeOrg, orgID)
		require.NoError(t, err)
		assert.Equal(t, "test-custom-role", roleDef.Name)
		assert.NotNil(t, roleDef.InheritsFrom)
		assert.Equal(t, models.RoleOrgViewer, roleDef.InheritsFrom.Name)
		assert.Len(t, roleDef.Permissions, 1)
	})

	t.Run("invalid request - missing role name", func(t *testing.T) {
		req := &pb.UpdateRoleRequest{
			RoleName:   "",
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		_, err := UpdateRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role name must be specified")
	})

	t.Run("invalid request - invalid domain type", func(t *testing.T) {
		req := &pb.UpdateRoleRequest{
			RoleName:   "test-custom-role",
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainId:   orgID,
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		_, err := UpdateRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid domain type")
	})

	t.Run("invalid request - default role name", func(t *testing.T) {
		req := &pb.UpdateRoleRequest{
			RoleName:   models.RoleOrgAdmin,
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		_, err := UpdateRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot update default role")
	})

	t.Run("invalid request - nonexistent role", func(t *testing.T) {
		req := &pb.UpdateRoleRequest{
			RoleName:   "nonexistent-role",
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		_, err := UpdateRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("invalid request - invalid UUID", func(t *testing.T) {
		req := &pb.UpdateRoleRequest{
			RoleName:   "test-custom-role",
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   "invalid-uuid",
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		_, err := UpdateRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UUIDs")
	})

	t.Run("invalid request - nonexistent inherited role", func(t *testing.T) {
		req := &pb.UpdateRoleRequest{
			RoleName:      "test-custom-role",
			DomainType:    pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:      orgID,
			InheritedRole: "nonexistent-role",
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		_, err := UpdateRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inherited role not found")
	})
}
