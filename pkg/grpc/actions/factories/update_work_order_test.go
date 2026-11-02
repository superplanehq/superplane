package factories

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

func Test__UpdateWorkOrder(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Old title", "Old body", &r.User, nil, nil)
	require.NoError(t, err)

	t.Run("updates title and description", func(t *testing.T) {
		title := "New title"
		description := "New body"
		resp, err := UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			OrderId:     order.ID.String(),
			Title:       &title,
			Description: &description,
		})
		require.NoError(t, err)
		assert.Equal(t, "New title", resp.Order.Title)
		assert.Equal(t, "New body", resp.Order.Description)

		refreshed, err := factoryModel.FindWorkOrder(database.DB(t.Context()), order.ID)
		require.NoError(t, err)
		assert.Equal(t, "New title", refreshed.Title)
		assert.Equal(t, "New body", refreshed.Description)
	})

	t.Run("updates description only", func(t *testing.T) {
		description := "Body only"
		resp, err := UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			OrderId:     order.ID.String(),
			Description: &description,
		})
		require.NoError(t, err)
		assert.Equal(t, "New title", resp.Order.Title)
		assert.Equal(t, "Body only", resp.Order.Description)
	})

	t.Run("rejects a blank title", func(t *testing.T) {
		title := "   "
		_, err := UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Title:     &title,
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("rejects a title that is too long", func(t *testing.T) {
		title := strings.Repeat("a", 257)
		_, err := UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Title:     &title,
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("rejects a request with no fields", func(t *testing.T) {
		_, err := UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})
}

func TestUpdateWorkOrderRestartsFailedPlanningAfterModelSave(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: models.FactoryAppBacklogTriggerID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "onWorkOrder"}}), Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateBacklogID, 1))},
	}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)
	order, err := factoryModel.CreateWorkOrder(db, "Plan this task", "", &r.User, []uuid.UUID{r.User}, nil)
	require.NoError(t, err)
	require.NoError(t, order.SetRefinementModel(db, "unavailable-model"))
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, models.FactoryAppBacklogTriggerID, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Create(&models.CanvasEvent{
		WorkflowID: canvas.ID, NodeID: models.FactoryAppBacklogTriggerID, RunID: run.ID,
		Data: models.NewJSONValue(map[string]any{"type": "workOrder.created", "data": map[string]any{"workOrder": map[string]any{"id": order.ID.String()}, models.WorkOrderCreatedRefinementEnabledDataKey: true}}),
	}).Error)
	require.NoError(t, db.Model(run).Updates(map[string]any{"state": models.CanvasRunStateFinished, "result": models.CanvasRunResultFailed}).Error)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	selected := "available-model"
	_, err = UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
		FactoryId: factoryModel.ID.String(), OrderId: order.ID.String(), RefinementModel: &selected,
	})
	require.NoError(t, err)
	var events []models.CanvasEvent
	require.NoError(t, db.Where("workflow_id = ? AND execution_id IS NULL", canvas.ID).Order("created_at ASC").Find(&events).Error)
	require.Len(t, events, 2, "the same task should get a new Backlog run")
	assert.NotEqual(t, run.ID, events[1].RunID)
	assert.Equal(t, order.ID.String(), models.RootEventSourcePayload(events[1].Data.Data()).(map[string]any)["workOrder"].(map[string]any)["id"])
	_, err = UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
		FactoryId: factoryModel.ID.String(), OrderId: order.ID.String(), RefinementModel: &selected,
	})
	require.NoError(t, err)
	otherModel := "another-model"
	_, err = UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
		FactoryId: factoryModel.ID.String(), OrderId: order.ID.String(), RefinementModel: &otherModel,
	})
	require.NoError(t, err)
	require.NoError(t, db.Where("workflow_id = ? AND execution_id IS NULL", canvas.ID).Order("created_at ASC").Find(&events).Error)
	assert.Len(t, events, 2, "saving again or changing models during an active run must not duplicate it")
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("id = ?", events[1].RunID).Updates(map[string]any{
		"state": models.CanvasRunStateFinished, "result": models.CanvasRunResultFailed,
	}).Error)
	_, err = UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
		FactoryId: factoryModel.ID.String(), OrderId: order.ID.String(), RefinementModel: &otherModel,
	})
	require.NoError(t, err)
	require.NoError(t, db.Where("workflow_id = ? AND execution_id IS NULL", canvas.ID).Order("created_at ASC").Find(&events).Error)
	assert.Len(t, events, 3, "retrying with the saved model must start a new run after a failure")
}
