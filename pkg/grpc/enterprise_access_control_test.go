package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

type fakeEntitlements map[licensing.Feature]bool

func (e fakeEntitlements) IsEntitled(feature licensing.Feature) bool {
	return e[feature]
}

func organizationContext(orgID, userID string) context.Context {
	ctx := authentication.SetUserIdInMetadata(context.Background(), userID)
	ctx = context.WithValue(ctx, authorization.OrganizationContextKey, orgID)
	ctx = context.WithValue(ctx, authorization.DomainTypeContextKey, models.DomainTypeOrganization)
	return context.WithValue(ctx, authorization.DomainIdContextKey, orgID)
}

func assertNotLicensed(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
	assert.Equal(t, licensing.ErrNotLicensed.Error(), grpcerrors.StatusMessage(err))
}

func TestCommunityAccessControlDeniesEnterpriseOperations(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()
	ctx := organizationContext(orgID, r.User.String())

	roles := NewRoleService(r.AuthService, licensing.Community, communityAccessControl{})
	groups := NewGroupsService(r.AuthService, communityAccessControl{})

	_, err := roles.CreateRole(ctx, &pbRoles.CreateRoleRequest{Role: &pbRoles.Role{
		Metadata: &pbRoles.Role_Metadata{Name: "custom-role"},
		Spec:     &pbRoles.Role_Spec{DisplayName: "Custom role"},
	}})
	assertNotLicensed(t, err)

	_, err = roles.UpdateRole(ctx, &pbRoles.UpdateRoleRequest{
		RoleName: "custom-role",
		Role:     &pbRoles.Role{Spec: &pbRoles.Role_Spec{DisplayName: "Custom role"}},
	})
	assertNotLicensed(t, err)

	_, err = groups.CreateGroup(ctx, &pbGroups.CreateGroupRequest{Group: &pbGroups.Group{
		Metadata: &pbGroups.Group_Metadata{Name: "engineers"},
		Spec:     &pbGroups.Group_Spec{Role: models.RoleOrgOperator},
	}})
	assertNotLicensed(t, err)

	_, err = groups.UpdateGroup(ctx, &pbGroups.UpdateGroupRequest{
		GroupName: "engineers",
		Group:     &pbGroups.Group{Spec: &pbGroups.Group_Spec{Role: models.RoleOrgOperator}},
	})
	assertNotLicensed(t, err)

	_, err = groups.AddUserToGroup(ctx, &pbGroups.AddUserToGroupRequest{
		GroupName: "engineers",
		UserId:    r.User.String(),
	})
	assertNotLicensed(t, err)
}

func TestNewServicesDefaultsToCommunity(t *testing.T) {
	r := support.Setup(t)
	ctx := organizationContext(r.Organization.ID.String(), r.User.String())

	services, err := NewServices(ServicesConfig{
		AuthService: r.AuthService,
		Registry:    r.Registry,
		Encryptor:   r.Encryptor,
		JWTSigner:   jwt.NewSigner("test-secret"),
	})
	require.NoError(t, err)

	_, err = services.Groups.CreateGroup(ctx, &pbGroups.CreateGroupRequest{Group: &pbGroups.Group{
		Metadata: &pbGroups.Group_Metadata{Name: "engineers"},
		Spec:     &pbGroups.Group_Spec{Role: models.RoleOrgOperator},
	}})
	assertNotLicensed(t, err)
}

func TestAssignRoleRequiresLicenseForCustomRoles(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()
	ctx := organizationContext(orgID, r.User.String())

	require.NoError(t, r.AuthService.CreateCustomRole(orgID, &authorization.RoleDefinition{
		Name:        "custom-role",
		DisplayName: "Custom role",
		DomainType:  models.DomainTypeOrganization,
		Permissions: []*authorization.Permission{
			{Resource: "canvases", Action: "read", DomainType: models.DomainTypeOrganization},
		},
	}))

	newUser := support.CreateUser(t, r, r.Organization.ID)

	t.Run("Community assigns built-in roles", func(t *testing.T) {
		roles := NewRoleService(r.AuthService, licensing.Community, communityAccessControl{})

		_, err := roles.AssignRole(ctx, &pbRoles.AssignRoleRequest{
			RoleName: models.RoleOrgOperator,
			UserId:   newUser.ID.String(),
		})
		require.NoError(t, err)
	})

	t.Run("Community rejects custom roles", func(t *testing.T) {
		roles := NewRoleService(r.AuthService, licensing.Community, communityAccessControl{})

		_, err := roles.AssignRole(ctx, &pbRoles.AssignRoleRequest{
			RoleName: "custom-role",
			UserId:   newUser.ID.String(),
		})
		assertNotLicensed(t, err)
	})

	t.Run("Enterprise assigns custom roles", func(t *testing.T) {
		roles := NewRoleService(
			r.AuthService,
			fakeEntitlements{licensing.FeatureCustomRoles: true},
			communityAccessControl{},
		)

		_, err := roles.AssignRole(ctx, &pbRoles.AssignRoleRequest{
			RoleName: "custom-role",
			UserId:   newUser.ID.String(),
		})
		require.NoError(t, err)
	})
}
