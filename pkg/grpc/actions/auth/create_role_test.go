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
				DisplayName: "Custom Role",
				Description: "Custom Role Description",
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

		response, err := DescribeRole(ctx, models.DomainTypeOrg, orgID, "custom-role", authService)
		require.NoError(t, err)
		createdRole := response.GetRole()
		assert.Equal(t, "custom-role", createdRole.GetMetadata().GetName())
		assert.Equal(t, "Custom Role", createdRole.GetSpec().GetDisplayName())
		assert.Equal(t, "Custom Role Description", createdRole.GetSpec().GetDescription())
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, createdRole.GetMetadata().GetDomainType())
		assert.Len(t, createdRole.GetSpec().GetPermissions(), 2)
	})

	t.Run("successful custom role creation with inheritance", func(t *testing.T) {
		role := &pb.Role{
			Metadata: &pb.Role_Metadata{
				Name: "custom-role-with-inheritance",
			},
			Spec: &pb.Role_Spec{
				DisplayName: "Custom Role With Inheritance",
				Description: "Custom Role With Inheritance Description",
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
		roleResponse, err := DescribeRole(ctx, models.DomainTypeOrg, orgID, "custom-role-with-inheritance", authService)
		require.NoError(t, err)
		createdRole := roleResponse.GetRole()
		assert.Equal(t, "custom-role-with-inheritance", createdRole.GetMetadata().GetName())
		assert.Equal(t, "Custom Role With Inheritance", createdRole.GetSpec().GetDisplayName())
		assert.Equal(t, "Custom Role With Inheritance Description", createdRole.GetSpec().GetDescription())
		assert.NotNil(t, createdRole.GetSpec().GetInheritedRole())
		assert.Equal(t, models.RoleOrgViewer, createdRole.GetSpec().GetInheritedRole().GetMetadata().GetName())
	})

	t.Run("invalid request - missing role name", func(t *testing.T) {
		role := &pb.Role{
			Metadata: &pb.Role_Metadata{
				Name: "",
			},
			Spec: &pb.Role_Spec{
				DisplayName: "Custom Role",
				Description: "Custom Role Description",
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
				DisplayName: "Custom Role",
				Description: "Custom Role Description",
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
				DisplayName: "Custom Role",
				Description: "Custom Role Description",
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
