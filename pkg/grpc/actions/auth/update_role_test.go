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
		req := &pb.Role_Spec{
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

		resp, err := UpdateRole(ctx, models.DomainTypeOrg, orgID, "test-custom-role", req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		roleDef, err := authService.GetRoleDefinition("test-custom-role", models.DomainTypeOrg, orgID)
		require.NoError(t, err)
		assert.Equal(t, "test-custom-role", roleDef.Name)
		assert.Len(t, roleDef.Permissions, 3)
	})

	t.Run("successful custom role update with inheritance", func(t *testing.T) {
		req := &pb.Role_Spec{
			InheritedRole: &pb.Role{
				Metadata: &pb.Role_Metadata{
					Name: models.RoleOrgViewer,
				},
			},
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "create",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		resp, err := UpdateRole(ctx, models.DomainTypeOrg, orgID, "test-custom-role", req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		roleDef, err := authService.GetRoleDefinition("test-custom-role", models.DomainTypeOrg, orgID)
		require.NoError(t, err)
		assert.Equal(t, "test-custom-role", roleDef.Name)
		assert.NotNil(t, roleDef.InheritsFrom)
		assert.Equal(t, models.RoleOrgViewer, roleDef.InheritsFrom.Name)
		assert.Len(t, roleDef.Permissions, 1)
	})

	t.Run("invalid request - missing role name", func(t *testing.T) {
		req := &pb.Role_Spec{
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		_, err := UpdateRole(ctx, models.DomainTypeOrg, orgID, "", req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role name must be specified")
	})

	t.Run("invalid request - default role name", func(t *testing.T) {
		req := &pb.Role_Spec{
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		_, err := UpdateRole(ctx, models.DomainTypeOrg, orgID, models.RoleOrgAdmin, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot update default role")
	})

	t.Run("invalid request - nonexistent role", func(t *testing.T) {
		req := &pb.Role_Spec{
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		_, err := UpdateRole(ctx, models.DomainTypeOrg, orgID, "nonexistent-role", req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("invalid request - nonexistent inherited role", func(t *testing.T) {
		req := &pb.Role_Spec{
			InheritedRole: &pb.Role{
				Metadata: &pb.Role_Metadata{
					Name: "nonexistent-role",
				},
			},
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		}

		_, err := UpdateRole(ctx, models.DomainTypeOrg, orgID, "test-custom-role", req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inherited role not found")
	})
}
