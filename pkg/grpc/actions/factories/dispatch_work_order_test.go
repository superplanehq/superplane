package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

// Test__DispatchWorkOrder__CreatesLineDispatchWithSnapshot covers acceptance
// criterion 1: dispatching a work order creates one line dispatch with the
// line's current steps snapshotted, and step 1's execution references it.
func Test__DispatchWorkOrder__CreatesLineDispatchWithSnapshot(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Name: "step-one", Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	require.Len(t, resp.Order.LineDispatches, 1)
	dispatch := resp.Order.LineDispatches[0]
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_ACTIVE, dispatch.State)
	assert.Equal(t, line.Name, dispatch.Line.Name)
	require.Len(t, dispatch.Steps, 1)
	assert.Equal(t, "step-one", dispatch.Steps[0].Name)
	require.Len(t, dispatch.StepExecutions, 1)
	assert.Equal(t, pb.WorkOrderExecution_STATE_PENDING, dispatch.StepExecutions[0].State)

	reloaded, err := models.FindUnscopedWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)

	active, err := order.FindActiveLineDispatch(db)
	require.NoError(t, err)
	assert.Equal(t, dispatch.Id, active.ID.String())
}

// Test__DispatchWorkOrder__RejectsWhenAlreadyActive covers acceptance
// criterion 5: a work order with an active line dispatch cannot be
// dispatched again.
func Test__DispatchWorkOrder__RejectsWhenAlreadyActive(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Name: "step-one", Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.Error(t, err)
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
}

// Test__DispatchWorkOrder__SecondDispatchAfterFirstFinishesCreatesSeparateTraversal
// covers acceptance criterion 4: dispatching a work order to the same line
// twice, after the first traversal finishes, produces two traversals shown
// separately by the API.
func Test__DispatchWorkOrder__SecondDispatchAfterFirstFinishesCreatesSeparateTraversal(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Name: "step-one", Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	firstResp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	require.Len(t, firstResp.Order.LineDispatches, 1)
	firstDispatchID := firstResp.Order.LineDispatches[0].Id

	firstDispatch, err := models.FindWorkOrderLineDispatch(db, uuid.MustParse(firstDispatchID))
	require.NoError(t, err)
	require.NoError(t, firstDispatch.Finish(db, models.CanvasRunResultFailed))

	secondResp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	require.Len(t, secondResp.Order.LineDispatches, 2,
		"two separate traversals of the same line, not one merged bucket")
	assert.Equal(t, firstDispatchID, secondResp.Order.LineDispatches[0].Id)
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_FINISHED, secondResp.Order.LineDispatches[0].State)
	assert.Equal(t, pb.WorkOrderLineDispatch_RESULT_FAILED, secondResp.Order.LineDispatches[0].Result)
	assert.NotEqual(t, firstDispatchID, secondResp.Order.LineDispatches[1].Id)
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_ACTIVE, secondResp.Order.LineDispatches[1].State)
}
