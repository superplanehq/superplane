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

func Test_UpdateRole(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create a custom role first
	customRoleDef := &authorization.RoleDefinition{
		Name:       "test-custom-role",
		DomainType: models.DomainTypeOrganization,
		Permissions: []*authorization.Permission{
			{
				Resource:   "canvas",
				Action:     "read",
				DomainType: models.DomainTypeOrganization,
			},
		},
	}
	err = authService.CreateCustomRole(orgID, customRoleDef)
	require.NoError(t, err)

	t.Run("successful custom role update", func(t *testing.T) {
		req := &pb.UpdateRoleRequest{
			RoleName:   "test-custom-role",
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			Permissions: []*pb.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
				{
					Resource:   "canvas",
					Action:     "write",
					DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
				{
					Resource:   "secret",
					Action:     "read",
					DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		resp, err := UpdateRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Check if role was updated by verifying permissions
		roleDef, err := authService.GetRoleDefinition("test-custom-role", models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, "test-custom-role", roleDef.Name)
		assert.Len(t, roleDef.Permissions, 3)
	})

	t.Run("successful custom role update with inheritance", func(t *testing.T) {
		req := &pb.UpdateRoleRequest{
			RoleName:      "test-custom-role",
			DomainType:    pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:      orgID,
			InheritedRole: authorization.RoleOrgViewer,
			Permissions: []*pb.Permission{
				{
					Resource:   "canvas",
					Action:     "create",
					DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		resp, err := UpdateRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Check if role was updated with inheritance
		roleDef, err := authService.GetRoleDefinition("test-custom-role", models.DomainTypeOrganization, orgID)
		require.NoError(t, err)
		assert.Equal(t, "test-custom-role", roleDef.Name)
		assert.NotNil(t, roleDef.InheritsFrom)
		assert.Equal(t, authorization.RoleOrgViewer, roleDef.InheritsFrom.Name)
		assert.Len(t, roleDef.Permissions, 1)
	})

	t.Run("invalid request - missing role name", func(t *testing.T) {
		req := &pb.UpdateRoleRequest{
			RoleName:   "",
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			Permissions: []*pb.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
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
			DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainId:   orgID,
			Permissions: []*pb.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		_, err := UpdateRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid domain type")
	})

	t.Run("invalid request - default role name", func(t *testing.T) {
		req := &pb.UpdateRoleRequest{
			RoleName:   authorization.RoleOrgAdmin,
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			Permissions: []*pb.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
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
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			Permissions: []*pb.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
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
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   "invalid-uuid",
			Permissions: []*pb.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
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
			DomainType:    pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:      orgID,
			InheritedRole: "nonexistent-role",
			Permissions: []*pb.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		_, err := UpdateRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inherited role not found")
	})
}
