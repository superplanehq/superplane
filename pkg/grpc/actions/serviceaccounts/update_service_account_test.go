package serviceaccounts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

func TestUpdateServiceAccountUpdatesCanvasScopeAndExpiration(t *testing.T) {
	r := support.Setup(t)
	firstCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	secondCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	existingExpiresAt := time.Now().Add(time.Hour)
	serviceAccount, err := models.CreateServiceAccount(
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
	response, err := UpdateServiceAccount(serviceAccountContext(r), &pb.UpdateServiceAccountRequest{
		Id:        serviceAccount.ID.String(),
		ExpiresAt: timestamppb.New(nextExpiresAt),
		CanvasIds: []string{secondCanvas.ID.String()},
	})
	require.NoError(t, err)
	require.Equal(t, []string{secondCanvas.ID.String()}, response.ServiceAccount.CanvasIds)
	require.Equal(t, nextExpiresAt.Unix(), response.ServiceAccount.ExpiresAt.AsTime().Unix())
}

func TestUpdateServiceAccountPreservesScopeWhenCanvasIdsOmitted(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	serviceAccount, err := models.CreateServiceAccount(
		database.Conn(),
		r.Organization.ID,
		"ci-bot",
		nil,
		r.User,
		nil,
		[]string{canvas.ID.String()},
	)
	require.NoError(t, err)

	_, err = UpdateServiceAccount(serviceAccountContext(r), &pb.UpdateServiceAccountRequest{
		Id:   serviceAccount.ID.String(),
		Name: "renamed-ci-bot",
	})
	require.NoError(t, err)

	var user models.User
	require.NoError(t, database.Conn().First(&user, "id = ?", serviceAccount.ID).Error)
	require.Equal(t, datatypes.NewJSONSlice([]string{canvas.ID.String()}), user.ServiceAccountCanvasIDs)
}

func TestUpdateServiceAccountClearsExpiration(t *testing.T) {
	r := support.Setup(t)
	expiresAt := time.Now().Add(time.Hour)
	serviceAccount, err := models.CreateServiceAccount(
		database.Conn(),
		r.Organization.ID,
		"ci-bot",
		nil,
		r.User,
		&expiresAt,
		nil,
	)
	require.NoError(t, err)

	response, err := UpdateServiceAccount(serviceAccountContext(r), &pb.UpdateServiceAccountRequest{
		Id:             serviceAccount.ID.String(),
		ClearExpiresAt: true,
	})
	require.NoError(t, err)
	require.Nil(t, response.ServiceAccount.ExpiresAt)
}
