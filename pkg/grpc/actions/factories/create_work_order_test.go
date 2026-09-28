package factories

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__CreateWorkOrder__AssignsTheCreator(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	other := support.CreateUser(t, r, r.Organization.ID)

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	resp, err := CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		Title:       "Ship the refunds line",
		AssigneeIds: []string{other.ID.String()},
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Assignees, 1)
	assert.Equal(t, r.User.String(), resp.Order.Assignees[0].Id)
	assert.Equal(t, r.UserModel.Name, resp.Order.Assignees[0].Name)
	assert.Equal(t, r.User.String(), resp.Order.GetCreatedBy().GetUser().GetId())
}

func Test__CreateWorkOrder__ReparentsWorkspaceFiles(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	file, err := models.CreatePendingFile(database.Conn(), models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(ctx, database.Conn(), store, file, bytes.NewReader([]byte("png-bytes"))))

	resp, err := CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		Title:       "Login screenshot",
		Description: "See ![bug](" + blob.FileRef(file.ID) + ")",
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Files, 1)
	assert.Equal(t, file.ID.String(), resp.Order.Files[0].Id)
	assert.NotEmpty(t, resp.Order.Files[0].DownloadUrl)
	assert.Contains(t, resp.Order.Description, blob.FileRef(file.ID))
	assert.NotContains(t, resp.Order.Description, "http://")

	reparented, err := models.FindFile(database.Conn(), file.ID)
	require.NoError(t, err)
	assert.Equal(t, blob.ScopeTask, reparented.Scope)
}

func Test__CreateWorkOrder__StoresAutoStartLine(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	resp, err := CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:       factoryModel.ID.String(),
		Title:           "Ship the refunds line",
		AutoStartLineId: line.ID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, line.ID.String(), resp.Order.GetAutoStartLineId())

	stored, err := models.FindUnscopedWorkOrder(db, uuid.MustParse(resp.Order.Id))
	require.NoError(t, err)
	require.NotNil(t, stored.AutoStartLineID)
	assert.Equal(t, line.ID, *stored.AutoStartLineID)
}

func Test__CreateWorkOrder__LeavesAutoStartOffWithoutLine(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	resp, err := CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		Title:     "Ship the refunds line",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Order.GetAutoStartLineId())

	stored, err := models.FindUnscopedWorkOrder(db, uuid.MustParse(resp.Order.Id))
	require.NoError(t, err)
	assert.Nil(t, stored.AutoStartLineID)
}

func Test__CreateWorkOrder__RejectsAutoStartLine(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	other, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("other"), "", "")
	require.NoError(t, err)
	foreignLine, err := other.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	_, err = CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:       factoryModel.ID.String(),
		Title:           "Ship the refunds line",
		AutoStartLineId: "not-a-uuid",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))

	_, err = CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:       factoryModel.ID.String(),
		Title:           "Ship the refunds line",
		AutoStartLineId: foreignLine.ID.String(),
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))

	planning := factoryModel.Planning()
	planning.Confidence = false
	require.NoError(t, factoryModel.UpdatePlanning(db, planning))
	line, err := factoryModel.CreateLine(db, "local", nil)
	require.NoError(t, err)
	_, err = CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:       factoryModel.ID.String(),
		Title:           "Ship the refunds line",
		AutoStartLineId: line.ID.String(),
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}
