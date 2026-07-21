package apikeys

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/api_keys"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

func TestUpdateAPIKeyUpdatesCanvasScopeAndExpiration(t *testing.T) {
	r := support.Setup(t)
	firstCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	secondCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	existingExpiresAt := time.Now().Add(time.Hour)
	apiKey, err := models.CreateAPIKey(
		database.Conn(),
		r.Organization.ID,
		"ci-bot",
		nil,
		r.User,
		&existingExpiresAt,
		[]string{firstCanvas.ID.String()},
	)
	require.NoError(t, err)

	nextExpiresAt := time.Now().Add(2 * time.Hour).UTC()
	response, err := UpdateAPIKey(apiKeyContext(r), &pb.UpdateAPIKeyRequest{
		Id:        apiKey.ID.String(),
		ExpiresAt: timestamppb.New(nextExpiresAt),
		CanvasIds: []string{secondCanvas.ID.String()},
	})
	require.NoError(t, err)
	require.Equal(t, []string{secondCanvas.ID.String()}, response.ApiKey.CanvasIds)
	require.Equal(t, nextExpiresAt.Unix(), response.ApiKey.ExpiresAt.AsTime().Unix())
}

func TestUpdateAPIKeyPreservesScopeWhenCanvasIdsOmitted(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	apiKey, err := models.CreateAPIKey(
		database.Conn(),
		r.Organization.ID,
		"ci-bot",
		nil,
		r.User,
		nil,
		[]string{canvas.ID.String()},
	)
	require.NoError(t, err)

	_, err = UpdateAPIKey(apiKeyContext(r), &pb.UpdateAPIKeyRequest{
		Id:   apiKey.ID.String(),
		Name: "renamed-ci-bot",
	})
	require.NoError(t, err)

	var user models.User
	require.NoError(t, database.Conn().First(&user, "id = ?", apiKey.ID).Error)
	require.Equal(t, datatypes.NewJSONSlice([]string{canvas.ID.String()}), user.APIKeyCanvasIDs)
}

func TestUpdateAPIKeyRejectsBlankName(t *testing.T) {
	r := support.Setup(t)
	apiKey, err := models.CreateAPIKey(
		database.Conn(),
		r.Organization.ID,
		"ci-bot",
		nil,
		r.User,
		nil,
		nil,
	)
	require.NoError(t, err)

	_, err = UpdateAPIKey(apiKeyContext(r), &pb.UpdateAPIKeyRequest{
		Id:   apiKey.ID.String(),
		Name: "   ",
	})
	require.Error(t, err)
}

func TestUpdateAPIKeyRejectsDuplicateName(t *testing.T) {
	r := support.Setup(t)
	_, err := models.CreateAPIKey(database.Conn(), r.Organization.ID, "ci-bot", nil, r.User, nil, nil)
	require.NoError(t, err)

	other, err := models.CreateAPIKey(database.Conn(), r.Organization.ID, "deploy-bot", nil, r.User, nil, nil)
	require.NoError(t, err)

	_, err = UpdateAPIKey(apiKeyContext(r), &pb.UpdateAPIKeyRequest{
		Id:   other.ID.String(),
		Name: "ci-bot",
	})
	require.Error(t, err)
	require.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
}

func TestUpdateAPIKeyClearsExpiration(t *testing.T) {
	r := support.Setup(t)
	expiresAt := time.Now().Add(time.Hour)
	apiKey, err := models.CreateAPIKey(
		database.Conn(),
		r.Organization.ID,
		"ci-bot",
		nil,
		r.User,
		&expiresAt,
		nil,
	)
	require.NoError(t, err)

	response, err := UpdateAPIKey(apiKeyContext(r), &pb.UpdateAPIKeyRequest{
		Id:             apiKey.ID.String(),
		ClearExpiresAt: true,
	})
	require.NoError(t, err)
	require.Nil(t, response.ApiKey.ExpiresAt)
}
