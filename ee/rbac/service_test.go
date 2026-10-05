package rbac

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

type fakeEntitlements map[licensing.Feature]bool

func (e fakeEntitlements) IsEntitled(feature licensing.Feature) bool {
	return e[feature]
}

func testRole(name string) *pbRoles.Role {
	return &pbRoles.Role{
		Metadata: &pbRoles.Role_Metadata{Name: name},
		Spec: &pbRoles.Role_Spec{
			DisplayName: name,
			Permissions: []*pbAuth.Permission{
				{
					Resource:   "canvases",
					Action:     "read",
					DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
				},
			},
		},
	}
}

func testGroup(name string) *pbGroups.Group {
	return &pbGroups.Group{
		Metadata: &pbGroups.Group_Metadata{Name: name},
		Spec: &pbGroups.Group_Spec{
			Role:        models.RoleOrgOperator,
			DisplayName: name,
		},
	}
}

func assertNotLicensed(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
	assert.Equal(t, licensing.ErrNotLicensed.Error(), grpcerrors.StatusMessage(err))
}

func Test_ServiceRequiresLicense(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()
	domainType := models.DomainTypeOrganization

	t.Run("Community denies every Enterprise operation", func(t *testing.T) {
		service := NewService(r.AuthService, licensing.Community)

		_, err := service.CreateRole(ctx, domainType, orgID, testRole("community-role"))
		assertNotLicensed(t, err)

		_, err = service.UpdateRole(ctx, domainType, orgID, "community-role", testRole("community-role").Spec)
		assertNotLicensed(t, err)

		_, err = service.CreateGroup(ctx, domainType, orgID, testGroup("community-group"))
		assertNotLicensed(t, err)

		_, err = service.UpdateGroup(ctx, domainType, orgID, "community-group", testGroup("community-group").Spec)
		assertNotLicensed(t, err)

		_, err = service.AddUserToGroup(ctx, orgID, domainType, orgID, r.User.String(), "", "community-group")
		assertNotLicensed(t, err)

		groups, err := r.AuthService.GetGroups(ctx, orgID, domainType)
		require.NoError(t, err)
		assert.NotContains(t, groups, "community-group")
	})

	t.Run("nil entitlements deny", func(t *testing.T) {
		service := NewService(r.AuthService, nil)

		_, err := service.CreateGroup(ctx, domainType, orgID, testGroup("nil-group"))
		assertNotLicensed(t, err)
	})

	t.Run("a feature does not unlock other features", func(t *testing.T) {
		service := NewService(r.AuthService, fakeEntitlements{licensing.FeatureGroups: true})

		_, err := service.CreateRole(ctx, domainType, orgID, testRole("groups-only-role"))
		assertNotLicensed(t, err)

		_, err = service.CreateGroup(ctx, domainType, orgID, testGroup("groups-only-group"))
		require.NoError(t, err)
	})

	t.Run("Enterprise allows licensed operations", func(t *testing.T) {
		service := NewService(r.AuthService, fakeEntitlements{
			licensing.FeatureCustomRoles: true,
			licensing.FeatureGroups:      true,
		})

		_, err := service.CreateRole(ctx, domainType, orgID, testRole("licensed-role"))
		require.NoError(t, err)

		_, err = service.UpdateRole(ctx, domainType, orgID, "licensed-role", testRole("licensed-role").Spec)
		require.NoError(t, err)

		_, err = service.CreateGroup(ctx, domainType, orgID, testGroup("licensed-group"))
		require.NoError(t, err)

		_, err = service.UpdateGroup(ctx, domainType, orgID, "licensed-group", testGroup("licensed-group").Spec)
		require.NoError(t, err)

		_, err = service.AddUserToGroup(ctx, orgID, domainType, orgID, r.User.String(), "", "licensed-group")
		require.NoError(t, err)
	})

	t.Run("a group cannot grant a custom role without the custom roles feature", func(t *testing.T) {
		service := NewService(r.AuthService, fakeEntitlements{licensing.FeatureGroups: true})

		group := testGroup("custom-role-group")
		group.Spec.Role = "licensed-role"
		_, err := service.CreateGroup(ctx, domainType, orgID, group)
		assertNotLicensed(t, err)

		_, err = service.UpdateGroup(ctx, domainType, orgID, "licensed-group", &pbGroups.Group_Spec{Role: "licensed-role"})
		assertNotLicensed(t, err)

		role, err := r.AuthService.GetGroupRole(ctx, orgID, domainType, "licensed-group")
		require.NoError(t, err)
		assert.Equal(t, models.RoleOrgOperator, role)
	})
}
