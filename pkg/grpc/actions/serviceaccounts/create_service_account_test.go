package serviceaccounts

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCreateServiceAccountStoresExpirationAndCanvasScope(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	expiresAt := time.Now().Add(time.Hour).UTC()

	response, err := CreateServiceAccount(serviceAccountContext(r), &pb.CreateServiceAccountRequest{
		Name:      "ci-bot",
		Role:      models.RoleOrgViewer,
		ExpiresAt: timestamppb.New(expiresAt),
		CanvasIds: []string{canvas.ID.String()},
	}, r.AuthService)
	require.NoError(t, err)
	require.NotNil(t, response.ServiceAccount)
	require.Equal(t, []string{canvas.ID.String()}, response.ServiceAccount.CanvasIds)
	require.Equal(t, expiresAt.Unix(), response.ServiceAccount.ExpiresAt.AsTime().Unix())

	var user models.User
	require.NoError(t, database.Conn().First(&user, "id = ?", response.ServiceAccount.Id).Error)
	require.Equal(t, []string{canvas.ID.String()}, []string(user.ServiceAccountCanvasIDs))
	require.NotNil(t, user.ServiceAccountExpiresAt)
	require.Equal(t, expiresAt.Unix(), user.ServiceAccountExpiresAt.Unix())
}

func TestCreateServiceAccountRejectsInvalidCanvasScope(t *testing.T) {
	r := support.Setup(t)

	_, err := CreateServiceAccount(serviceAccountContext(r), &pb.CreateServiceAccountRequest{
		Name:      "ci-bot",
		Role:      models.RoleOrgViewer,
		CanvasIds: []string{"not-a-canvas-id"},
	}, r.AuthService)

	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func serviceAccountContext(r *support.ResourceRegistry) context.Context {
	return metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(
			"x-organization-id", r.Organization.ID.String(),
			"x-user-id", r.User.String(),
		),
	)
}
