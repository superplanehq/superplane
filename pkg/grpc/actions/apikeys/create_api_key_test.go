package apikeys

import (
	"context"
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

func TestCreateAPIKeyAllowsCustomRole(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()

	customRole := &authorization.RoleDefinition{
		Name:       "ci-custom-role",
		DomainType: models.DomainTypeOrganization,
		Permissions: []*authorization.Permission{
			{
				Resource:   "canvases",
				Action:     "read",
				DomainType: models.DomainTypeOrganization,
			},
		},
	}
	require.NoError(t, r.AuthService.CreateCustomRole(orgID, customRole))

	response, err := CreateAPIKey(apiKeyContext(r), &pb.CreateAPIKeyRequest{
		Name: "ci-bot",
		Role: "ci-custom-role",
	}, r.AuthService)
	require.NoError(t, err)
	require.NotNil(t, response.ApiKey)

	roles, err := r.AuthService.GetUserRolesForOrg(context.Background(), response.ApiKey.Id, orgID)
	require.NoError(t, err)
	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}
	require.Contains(t, roleNames, "ci-custom-role")
}

func TestCreateAPIKeyRejectsOrgOwnerRole(t *testing.T) {
	r := support.Setup(t)

	_, err := CreateAPIKey(apiKeyContext(r), &pb.CreateAPIKeyRequest{
		Name: "ci-bot",
		Role: models.RoleOrgOwner,
	}, r.AuthService)

	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func TestCreateAPIKeyRejectsUnknownRole(t *testing.T) {
	r := support.Setup(t)

	_, err := CreateAPIKey(apiKeyContext(r), &pb.CreateAPIKeyRequest{
		Name: "ci-bot",
		Role: "does-not-exist",
	}, r.AuthService)

	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
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
