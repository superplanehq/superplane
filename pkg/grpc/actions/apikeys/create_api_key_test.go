package apikeys

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/api_keys"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func TestCreateAPIKeyStoresExpirationAndCanvasScope(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	expiresAt := time.Now().Add(time.Hour).UTC()

	response, err := CreateAPIKey(apiKeyContext(r), &pb.CreateAPIKeyRequest{
		Name:      "ci-bot",
		Role:      models.RoleOrgViewer,
		ExpiresAt: timestamppb.New(expiresAt),
		CanvasIds: []string{canvas.ID.String()},
	}, r.AuthService)
	require.NoError(t, err)
	require.NotNil(t, response.ApiKey)
	require.Equal(t, []string{canvas.ID.String()}, response.ApiKey.CanvasIds)
	require.Equal(t, expiresAt.Unix(), response.ApiKey.ExpiresAt.AsTime().Unix())

	var user models.User
	require.NoError(t, database.Conn().First(&user, "id = ?", response.ApiKey.Id).Error)
	require.Equal(t, []string{canvas.ID.String()}, []string(user.APIKeyCanvasIDs))
	require.NotNil(t, user.APIKeyExpiresAt)
	require.Equal(t, expiresAt.Unix(), user.APIKeyExpiresAt.Unix())
}

func TestCreateAPIKeyRejectsInvalidCanvasScope(t *testing.T) {
	r := support.Setup(t)

	_, err := CreateAPIKey(apiKeyContext(r), &pb.CreateAPIKeyRequest{
		Name:      "ci-bot",
		Role:      models.RoleOrgViewer,
		CanvasIds: []string{"not-a-canvas-id"},
	}, r.AuthService)

	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func TestCreateAPIKeyAcceptsCustomRole(t *testing.T) {
	r := support.Setup(t)
	roleName := "ci-deployer"
	require.NoError(t, r.AuthService.CreateCustomRole(r.Organization.ID.String(), &authorization.RoleDefinition{
		Name:       roleName,
		DomainType: models.DomainTypeOrganization,
		Permissions: []*authorization.Permission{
			{Resource: "canvases", Action: "create", DomainType: models.DomainTypeOrganization},
		},
	}))

	response, err := CreateAPIKey(apiKeyContext(r), &pb.CreateAPIKeyRequest{
		Name: "ci-bot",
		Role: roleName,
	}, r.AuthService)
	require.NoError(t, err)
	require.NotEmpty(t, response.Token)
	require.EqualValues(t, 1, countAPIKeyRoleAssignment(t, response.ApiKey.Id, roleName, r.Organization.ID.String()))
}

func TestCreateAPIKeyAcceptsInheritanceOnlyCustomRole(t *testing.T) {
	r := support.Setup(t)
	roleName := "read-only-agent"
	require.NoError(t, r.AuthService.CreateCustomRole(r.Organization.ID.String(), &authorization.RoleDefinition{
		Name:         roleName,
		DomainType:   models.DomainTypeOrganization,
		InheritsFrom: &authorization.RoleDefinition{Name: models.RoleOrgViewer},
	}))

	response, err := CreateAPIKey(apiKeyContext(r), &pb.CreateAPIKeyRequest{
		Name: "read-only-bot",
		Role: roleName,
	}, r.AuthService)
	require.NoError(t, err)
	require.EqualValues(t, 1, countAPIKeyRoleAssignment(t, response.ApiKey.Id, roleName, r.Organization.ID.String()))
}

func TestCreateAPIKeyRejectsUnavailableRoles(t *testing.T) {
	tests := []struct {
		name string
		role string
	}{
		{name: "blank", role: "   "},
		{name: "owner", role: models.RoleOrgOwner},
		{name: "unknown", role: "missing-role"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := support.Setup(t)
			_, err := CreateAPIKey(apiKeyContext(r), &pb.CreateAPIKeyRequest{
				Name: "invalid-role-bot",
				Role: tt.role,
			}, r.AuthService)

			require.Error(t, err)
			require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
			require.Zero(t, countAPIKeys(t, r.Organization.ID.String()))
		})
	}
}

func TestCreateAPIKeyRejectsOwnerDerivedCustomRole(t *testing.T) {
	r := support.Setup(t)
	roleName := "owner-derived"
	require.NoError(t, r.AuthService.CreateCustomRole(r.Organization.ID.String(), &authorization.RoleDefinition{
		Name:         roleName,
		DomainType:   models.DomainTypeOrganization,
		InheritsFrom: &authorization.RoleDefinition{Name: models.RoleOrgOwner},
	}))

	_, err := CreateAPIKey(apiKeyContext(r), &pb.CreateAPIKeyRequest{
		Name: "owner-derived-bot",
		Role: roleName,
	}, r.AuthService)

	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	require.Zero(t, countAPIKeys(t, r.Organization.ID.String()))
}

func TestCreateAPIKeyRollsBackWhenRoleAssignmentFails(t *testing.T) {
	r := support.Setup(t)
	authService := &failingAssignRoleAuthorization{
		Authorization: r.AuthService,
		err:           errors.New("authorization database unavailable"),
	}

	_, err := CreateAPIKey(apiKeyContext(r), &pb.CreateAPIKeyRequest{
		Name: "rollback-bot",
		Role: models.RoleOrgViewer,
	}, authService)

	require.Error(t, err)
	require.Equal(t, codes.Internal, grpcerrors.Code(err))
	require.Zero(t, countAPIKeys(t, r.Organization.ID.String()))
}

type failingAssignRoleAuthorization struct {
	authorization.Authorization
	err error
}

func (a *failingAssignRoleAuthorization) AssignRole(_ *gorm.DB, _, _, _, _ string) error {
	return a.err
}

func countAPIKeyRoleAssignment(t *testing.T, userID, roleName, orgID string) int64 {
	t.Helper()

	var count int64
	require.NoError(t, database.DB(t.Context()).Table("casbin_rule").
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "g", "/users/"+userID, "/roles/"+roleName, "/org/"+orgID).
		Count(&count).
		Error)

	return count
}

func countAPIKeys(t *testing.T, orgID string) int64 {
	t.Helper()

	var count int64
	require.NoError(t, database.DB(t.Context()).Unscoped().Model(&models.User{}).
		Where("organization_id = ? AND type = ?", orgID, models.UserTypeAPIKey).
		Count(&count).
		Error)

	return count
}

func apiKeyContext(r *support.ResourceRegistry) context.Context {
	return metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(
			"x-organization-id", r.Organization.ID.String(),
			"x-user-id", r.User.String(),
		),
	)
}
