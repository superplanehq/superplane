package factories

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
)

func Test__CreateWorkOrder__RefinementModel(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider: models.UsageProviderAnthropic, Enabled: true, APIKey: []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6", "claude-opus-4-6"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{}) })
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "trigger", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "onWorkOrder"}}), Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateBacklogID, 3))},
		{NodeID: "refine-task", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}}), Configuration: datatypes.NewJSONType(map[string]any{"model": "claude-sonnet-4-6", "credentials": map[string]any{"source": "hosted"}})},
	}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", f.ID).Error)
	for _, model := range []string{"", "claude-opus-4-6", "gpt-5"} {
		t.Run(model, func(t *testing.T) {
			resp, err := CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{FactoryId: f.ID.String(), Title: "Refine this task", Model: model})
			if model == "gpt-5" {
				code, _, ok := grpcerrors.HandlerStatus(err)
				require.True(t, ok)
				require.Equal(t, codes.InvalidArgument, code)
				var count int64
				require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("factory_id = ?", f.ID).Count(&count).Error)
				assert.Equal(t, int64(2), count)
				return
			}
			require.NoError(t, err)
			order, err := f.FindWorkOrder(db, uuid.MustParse(resp.Order.Id))
			require.NoError(t, err)
			if model == "" {
				assert.Nil(t, order.RefinementModel)
			} else {
				require.NotNil(t, order.RefinementModel)
				assert.Equal(t, model, *order.RefinementModel)
			}
		})
	}
	node, err := models.FindCanvasNode(db, canvas.ID, "refine-task")
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet-4-6", node.Configuration.Data()["model"])
}

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
