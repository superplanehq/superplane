package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
)

func Test_CreateRole(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful custom role creation", func(t *testing.T) {
		req := &pb.CreateRoleRequest{
			Name:       "custom-role",
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
			},
		}

		resp, err := CreateRole(ctx, models.DomainTypeOrg, orgID, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Check if role was created by verifying we can get its definition
		roleDef, err := authService.GetRoleDefinition("custom-role", models.DomainTypeOrg, orgID)
		require.NoError(t, err)
		assert.Equal(t, "custom-role", roleDef.Name)
		assert.Equal(t, models.DomainTypeOrg, roleDef.DomainType)
		assert.Len(t, roleDef.Permissions, 2)
	})

	t.Run("successful custom role creation with inheritance", func(t *testing.T) {
		req := &pb.CreateRoleRequest{
			Name:          "custom-role-with-inheritance",
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

		resp, err := CreateRole(ctx, models.DomainTypeOrg, orgID, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Check if role was created with inheritance
		roleDef, err := authService.GetRoleDefinition("custom-role-with-inheritance", models.DomainTypeOrg, orgID)
		require.NoError(t, err)
		assert.Equal(t, "custom-role-with-inheritance", roleDef.Name)
		assert.NotNil(t, roleDef.InheritsFrom)
		assert.Equal(t, models.RoleOrgViewer, roleDef.InheritsFrom.Name)
	})

	t.Run("invalid request - missing role name", func(t *testing.T) {
		req := &pb.CreateRoleRequest{
			Name:       "",
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

		_, err := CreateRole(ctx, models.DomainTypeOrg, orgID, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role name must be specified")
	})

	t.Run("invalid request - default role name", func(t *testing.T) {
		req := &pb.CreateRoleRequest{
			Name:       models.RoleOrgAdmin,
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

		_, err := CreateRole(ctx, models.DomainTypeOrg, orgID, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot create custom role with default role name")
	})

	t.Run("invalid request - nonexistent inherited role", func(t *testing.T) {
		req := &pb.CreateRoleRequest{
			Name:          "test-role",
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

		_, err := CreateRole(ctx, models.DomainTypeOrg, orgID, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inherited role not found")
	})
}
