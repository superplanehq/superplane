package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test_RolePrivilegeEscalationGuards(t *testing.T) {
	r := support.Setup(t)
	ownerCtx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	admin := support.CreateUser(t, r, r.Organization.ID)
	_, err := AssignRole(ownerCtx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, admin.ID.String(), "", r.AuthService)
	require.NoError(t, err)
	adminCtx := authentication.SetUserIdInMetadata(context.Background(), admin.ID.String())

	t.Run("admin cannot create group with org_owner role", func(t *testing.T) {
		_, err := CreateGroup(adminCtx, models.DomainTypeOrganization, orgID, &pb.Group{
			Metadata: &pb.Group_Metadata{Name: "owner-group"},
			Spec:     &pb.Group_Spec{Role: models.RoleOrgOwner, DisplayName: "Owner Group"},
		}, r.AuthService)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, code)
		assert.Equal(t, "cannot assign a role with permissions you do not have", msg)
	})

	t.Run("admin cannot update group role to org_owner", func(t *testing.T) {
		_, err := CreateGroup(ownerCtx, models.DomainTypeOrganization, orgID, &pb.Group{
			Metadata: &pb.Group_Metadata{Name: "admin-group"},
			Spec:     &pb.Group_Spec{Role: models.RoleOrgAdmin, DisplayName: "Admin Group"},
		}, r.AuthService)
		require.NoError(t, err)

		_, err = UpdateGroup(adminCtx, models.DomainTypeOrganization, orgID, "admin-group", &pb.Group_Spec{
			Role: models.RoleOrgOwner,
		}, r.AuthService)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, code)
		assert.Equal(t, "cannot assign a role with permissions you do not have", msg)
	})

	t.Run("admin cannot add user to org_owner group", func(t *testing.T) {
		_, err := CreateGroup(ownerCtx, models.DomainTypeOrganization, orgID, &pb.Group{
			Metadata: &pb.Group_Metadata{Name: "owners"},
			Spec:     &pb.Group_Spec{Role: models.RoleOrgOwner, DisplayName: "Owners"},
		}, r.AuthService)
		require.NoError(t, err)

		target := support.CreateUser(t, r, r.Organization.ID)
		_, err = AddUserToGroup(adminCtx, orgID, models.DomainTypeOrganization, orgID, target.ID.String(), "", "owners", r.AuthService)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, code)
		assert.Equal(t, "cannot assign a role with permissions you do not have", msg)
	})

	t.Run("admin cannot create role with owner-only permission", func(t *testing.T) {
		_, err := CreateRole(adminCtx, models.DomainTypeOrganization, orgID, &pbRoles.Role{
			Metadata: &pbRoles.Role_Metadata{Name: "almost-owner"},
			Spec: &pbRoles.Role_Spec{
				DisplayName: "Almost Owner",
				Permissions: []*pbAuth.Permission{
					{Resource: "org", Action: "delete", DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION},
				},
			},
		}, r.AuthService)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, code)
		assert.Equal(t, "cannot assign a role with permissions you do not have", msg)
	})

	t.Run("admin cannot create role inheriting org_owner", func(t *testing.T) {
		_, err := CreateRole(adminCtx, models.DomainTypeOrganization, orgID, &pbRoles.Role{
			Metadata: &pbRoles.Role_Metadata{Name: "inherits-owner"},
			Spec: &pbRoles.Role_Spec{
				DisplayName: "Inherits Owner",
				Permissions: []*pbAuth.Permission{
					{Resource: "canvases", Action: "read", DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION},
				},
				InheritedRole: &pbRoles.Role{
					Metadata: &pbRoles.Role_Metadata{Name: models.RoleOrgOwner},
				},
			},
		}, r.AuthService)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, code)
		assert.Equal(t, "cannot assign a role with permissions you do not have", msg)
	})

	t.Run("admin can create role with admin-level permissions", func(t *testing.T) {
		resp, err := CreateRole(adminCtx, models.DomainTypeOrganization, orgID, &pbRoles.Role{
			Metadata: &pbRoles.Role_Metadata{Name: "canvas-editor"},
			Spec: &pbRoles.Role_Spec{
				DisplayName: "Canvas Editor",
				Permissions: []*pbAuth.Permission{
					{Resource: "canvases", Action: "read", DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION},
					{Resource: "canvases", Action: "update", DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION},
				},
			},
		}, r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}
