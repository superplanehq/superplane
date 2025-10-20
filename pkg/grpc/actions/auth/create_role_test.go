package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/metadata"
)

func Test_CreateRole(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()
	ctx = authentication.SetOrganizationIdInMetadata(ctx, orgID)

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

		resp, err := CreateRole(ctx, models.DomainTypeOrganization, orgID, role, r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		response, err := DescribeRole(ctx, models.DomainTypeOrganization, orgID, "custom-role", r.AuthService)
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

		resp, err := CreateRole(ctx, models.DomainTypeOrganization, orgID, role, r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Check if role was created with inheritance
		roleResponse, err := DescribeRole(ctx, models.DomainTypeOrganization, orgID, "custom-role-with-inheritance", r.AuthService)
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

		_, err := CreateRole(ctx, models.DomainTypeOrganization, orgID, role, r.AuthService)
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

		_, err := CreateRole(ctx, models.DomainTypeOrganization, orgID, role, r.AuthService)
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

		_, err := CreateRole(ctx, models.DomainTypeOrganization, orgID, role, r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inherited role not found")
	})

	t.Run("create global canvas role with organization context", func(t *testing.T) {
		err := r.AuthService.SetupGlobalCanvasRoles(orgID)
		require.NoError(t, err)

		md := metadata.Pairs("x-organization-id", orgID)
		ctxWithOrg := metadata.NewIncomingContext(ctx, md)

		role := &pb.Role{
			Metadata: &pb.Role_Metadata{
				Name: "global-canvas-custom-role",
			},
			Spec: &pb.Role_Spec{
				DisplayName: "Global Canvas Custom Role",
				Description: "Custom role for global canvas operations",
				Permissions: []*pbAuth.Permission{
					{
						Resource:   "canvas",
						Action:     "read",
						DomainType: pbAuth.DomainType_DOMAIN_TYPE_CANVAS,
					},
					{
						Resource:   "canvas",
						Action:     "write",
						DomainType: pbAuth.DomainType_DOMAIN_TYPE_CANVAS,
					},
				},
			},
		}

		resp, err := CreateRole(ctxWithOrg, models.DomainTypeCanvas, "*", role, r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		retrievedRole, err := r.AuthService.GetGlobalCanvasRoleDefinition("global-canvas-custom-role", orgID)
		require.NoError(t, err)
		assert.Equal(t, "global-canvas-custom-role", retrievedRole.Name)
		assert.Equal(t, "Global Canvas Custom Role", retrievedRole.DisplayName)
		assert.Equal(t, "Custom role for global canvas operations", retrievedRole.Description)
		assert.Equal(t, models.DomainTypeCanvas, retrievedRole.DomainType)
		assert.Len(t, retrievedRole.Permissions, 2)
		assert.False(t, retrievedRole.Readonly)
	})
}
