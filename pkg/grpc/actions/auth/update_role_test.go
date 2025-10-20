package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/metadata"
)

func Test_UpdateRole(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

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
	err := r.AuthService.CreateCustomRole(orgID, customRoleDef)
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

		resp, err := UpdateRole(ctx, models.DomainTypeOrganization, orgID, "test-custom-role", req, r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		roleDef, err := r.AuthService.GetRoleDefinition("test-custom-role", models.DomainTypeOrganization, orgID)
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

		resp, err := UpdateRole(ctx, models.DomainTypeOrganization, orgID, "test-custom-role", req, r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		roleDef, err := r.AuthService.GetRoleDefinition("test-custom-role", models.DomainTypeOrganization, orgID)
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

		_, err := UpdateRole(ctx, models.DomainTypeOrganization, orgID, "", req, r.AuthService)
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

		_, err := UpdateRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, req, r.AuthService)
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

		_, err := UpdateRole(ctx, models.DomainTypeOrganization, orgID, "nonexistent-role", req, r.AuthService)
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

		_, err := UpdateRole(ctx, models.DomainTypeOrganization, orgID, "test-custom-role", req, r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inherited role not found")
	})

	t.Run("update global canvas role with organization context", func(t *testing.T) {
		err := r.AuthService.SetupGlobalCanvasRoles(orgID)
		require.NoError(t, err)

		md := metadata.Pairs("x-organization-id", orgID)
		ctxWithOrg := metadata.NewIncomingContext(ctx, md)

		globalRoleDef := &authorization.RoleDefinition{
			Name:        "global-canvas-updatable-role",
			DisplayName: "Global Canvas Updatable Role",
			Description: "Role to be updated",
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

		updateReq := &pb.Role_Spec{
			DisplayName: "Updated Global Canvas Role",
			Description: "Updated description for global canvas role",
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
				{
					Resource:   "canvas",
					Action:     "delete",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_CANVAS,
				},
			},
		}

		resp, err := UpdateRole(ctxWithOrg, models.DomainTypeCanvas, "*", "global-canvas-updatable-role", updateReq, r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		retrievedRole, err := r.AuthService.GetGlobalCanvasRoleDefinition("global-canvas-updatable-role", orgID)
		require.NoError(t, err)
		assert.Equal(t, "global-canvas-updatable-role", retrievedRole.Name)
		assert.Equal(t, "Updated Global Canvas Role", retrievedRole.DisplayName)
		assert.Equal(t, "Updated description for global canvas role", retrievedRole.Description)
		assert.Equal(t, models.DomainTypeCanvas, retrievedRole.DomainType)
		assert.Len(t, retrievedRole.Permissions, 3)
		assert.False(t, retrievedRole.Readonly)
	})

	t.Run("update global canvas role without organization context fails", func(t *testing.T) {
		// Try to update without organization context
		updateReq := &pb.Role_Spec{
			DisplayName: "Should Fail Update",
			Description: "Should fail without organization context",
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvas",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_CANVAS,
				},
			},
		}

		_, err := UpdateRole(ctx, models.DomainTypeCanvas, "*", "global-canvas-updatable-role", updateReq, r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "organization context required for global canvas roles")
	})
}
