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

func Test__DuplicateWorkOrder(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	t.Run("copies title description and origin into a new draft", func(t *testing.T) {
		author := support.CreateUser(t, r, r.Organization.ID)
		source, err := factoryModel.CreateWorkOrderWithOrigin(
			db,
			"Retry refunds",
			"Stop double charges.",
			&author.ID,
			[]uuid.UUID{author.ID},
			nil,
			models.WorkOrderOrigin{URL: "https://github.com/acme/api/issues/42", Label: "acme/api#42"},
		)
		require.NoError(t, err)

		resp, err := DuplicateWorkOrder(ctx, r.Organization.ID.String(), &pb.DuplicateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   source.ID.String(),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Order)
		assert.NotEqual(t, source.ID.String(), resp.Order.Id)
		assert.NotEqual(t, source.Number, resp.Order.Number)
		assert.Equal(t, "Retry refunds", resp.Order.Title)
		assert.Equal(t, "Stop double charges.", resp.Order.Description)
		assert.Equal(t, pb.WorkOrder_STATE_DRAFT, resp.Order.State)
		assert.Equal(t, pb.WorkOrder_RESULT_UNSPECIFIED, resp.Order.Result)
		assert.Equal(t, r.User.String(), resp.Order.GetCreatedBy().GetUser().GetId())
		require.Len(t, resp.Order.Assignees, 1)
		assert.Equal(t, r.User.String(), resp.Order.Assignees[0].Id)
		require.NotNil(t, resp.Order.Origin)
		assert.Equal(t, "https://github.com/acme/api/issues/42", resp.Order.Origin.Url)
		assert.Equal(t, "acme/api#42", resp.Order.Origin.Label)
		assert.Empty(t, resp.Order.LineDispatches)
	})

	t.Run("clones description files and leaves the source unchanged", func(t *testing.T) {
		t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
		t.Setenv("BASE_URL", "http://files.test")
		store, err := filesystem.New(t.TempDir())
		require.NoError(t, err)
		blob.SetCurrent(store)
		t.Cleanup(func() { blob.SetCurrent(nil) })

		file, err := models.CreatePendingFile(db, models.CreateFileParams{
			Scope:          blob.ScopeWorkspace,
			OrganizationID: r.Organization.ID,
			FactoryID:      factoryModel.ID,
			Filename:       "bug.png",
			ContentType:    "image/png",
			CreatedByID:    r.User,
		})
		require.NoError(t, err)
		require.NoError(t, storedfiles.CompleteUpload(t.Context(), db, store, file, bytes.NewReader([]byte("png-bytes"))))

		source, err := CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			Title:       "Login screenshot",
			Description: "See ![bug](" + blob.FileRef(file.ID) + ")",
		})
		require.NoError(t, err)
		require.Len(t, source.Order.Files, 1)
		sourceFileID := source.Order.Files[0].Id

		resp, err := DuplicateWorkOrder(ctx, r.Organization.ID.String(), &pb.DuplicateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   source.Order.Id,
		})
		require.NoError(t, err)
		require.Len(t, resp.Order.Files, 1)
		assert.NotEqual(t, sourceFileID, resp.Order.Files[0].Id)
		assert.Contains(t, resp.Order.Description, blob.FileRef(uuid.MustParse(resp.Order.Files[0].Id)))
		assert.NotContains(t, resp.Order.Description, blob.FileRef(uuid.MustParse(sourceFileID)))

		sourceOrderID, err := uuid.Parse(source.Order.Id)
		require.NoError(t, err)
		sourceFiles, err := models.ListReadyTaskFiles(db, sourceOrderID)
		require.NoError(t, err)
		require.Len(t, sourceFiles, 1)
		assert.Equal(t, sourceFileID, sourceFiles[0].ID.String())
	})

	t.Run("duplicates an empty description without files", func(t *testing.T) {
		source, err := factoryModel.CreateWorkOrder(db, "Empty body", "", &r.User, nil, nil)
		require.NoError(t, err)

		resp, err := DuplicateWorkOrder(ctx, r.Organization.ID.String(), &pb.DuplicateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   source.ID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, "Empty body", resp.Order.Title)
		assert.Empty(t, resp.Order.Description)
		assert.Empty(t, resp.Order.Files)
		assert.Equal(t, pb.WorkOrder_STATE_DRAFT, resp.Order.State)
	})

	t.Run("returns not found for an unknown factory", func(t *testing.T) {
		_, err := DuplicateWorkOrder(ctx, r.Organization.ID.String(), &pb.DuplicateWorkOrderRequest{
			FactoryId: uuid.New().String(),
			OrderId:   uuid.New().String(),
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("returns not found for an unknown order", func(t *testing.T) {
		_, err := DuplicateWorkOrder(ctx, r.Organization.ID.String(), &pb.DuplicateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   uuid.New().String(),
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		source, err := factoryModel.CreateWorkOrder(db, "Auth", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = DuplicateWorkOrder(context.Background(), r.Organization.ID.String(), &pb.DuplicateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   source.ID.String(),
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
	})
}
