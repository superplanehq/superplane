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
		role := &pb.Role{
			Metadata: &pb.Role_Metadata{
				Name: "custom-role",
			},
			Spec: &pb.Role_Spec{
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
			},
		}

		resp, err := CreateRole(ctx, models.DomainTypeOrg, orgID, role, authService)
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
		role := &pb.Role{
			Metadata: &pb.Role_Metadata{
				Name: "custom-role-with-inheritance",
			},
			Spec: &pb.Role_Spec{
				Permissions: []*pbAuth.Permission{
					{
						Resource:   "canvas",
						Action:     "create",
						DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
					},
				},
				InheritedRole: &pb.Role{
					Metadata: &pb.Role_Metadata{
						Name: models.RoleOrgViewer,
					},
				},
			},
		}

		resp, err := CreateRole(ctx, models.DomainTypeOrg, orgID, role, authService)
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
		role := &pb.Role{
			Metadata: &pb.Role_Metadata{
				Name: "",
			},
			Spec: &pb.Role_Spec{
				Permissions: []*pbAuth.Permission{
					{
						Resource:   "canvas",
						Action:     "read",
						DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
					},
				},
			},
		}

		_, err := CreateRole(ctx, models.DomainTypeOrg, orgID, role, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role name must be specified")
	})

	t.Run("invalid request - default role name", func(t *testing.T) {
		role := &pb.Role{
			Metadata: &pb.Role_Metadata{
				Name: models.RoleOrgAdmin,
			},
			Spec: &pb.Role_Spec{
				Permissions: []*pbAuth.Permission{
					{
						Resource:   "canvas",
						Action:     "read",
						DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
					},
				},
			},
		}

		_, err := CreateRole(ctx, models.DomainTypeOrg, orgID, role, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot create custom role with default role name")
	})

	t.Run("invalid request - nonexistent inherited role", func(t *testing.T) {
		role := &pb.Role{
			Metadata: &pb.Role_Metadata{
				Name: "test-role",
			},
			Spec: &pb.Role_Spec{
				Permissions: []*pbAuth.Permission{
					{
						Resource:   "canvas",
						Action:     "read",
						DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
					},
				},
				InheritedRole: &pb.Role{
					Metadata: &pb.Role_Metadata{
						Name: "nonexistent-role",
					},
				},
			},
		}

		_, err := CreateRole(ctx, models.DomainTypeOrg, orgID, role, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inherited role not found")
	})
}
