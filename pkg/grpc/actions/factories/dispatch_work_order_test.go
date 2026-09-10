package factories

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func Test__DispatchWorkOrder__RetryReturnsExistingDispatch(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	key := "dispatch-retry"
	request := &pb.DispatchWorkOrderRequest{
		FactoryId:      factoryModel.ID.String(),
		OrderId:        order.ID.String(),
		LineName:       line.Name,
		IdempotencyKey: &key,
	}
	first, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), request)
	require.NoError(t, err)
	second, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), request)
	require.NoError(t, err)

	require.Len(t, first.Order.LineDispatches, 1)
	require.Len(t, second.Order.LineDispatches, 1)
	assert.Equal(t, first.Order.LineDispatches[0].Id, second.Order.LineDispatches[0].Id)

	var dispatches, executions, events int64
	require.NoError(t, db.Model(&models.FactoryWorkOrderLineDispatch{}).Where("work_order_id = ?", order.ID).Count(&dispatches).Error)
	require.NoError(t, db.Model(&models.FactoryWorkOrderExecution{}).Where("work_order_id = ?", order.ID).Count(&executions).Error)
	require.NoError(t, db.Model(&models.FactoryWorkOrderEvent{}).
		Where("work_order_id = ? AND type = ?", order.ID, factoryevents.EventTypeLineStepExecutionCreated).
		Count(&events).Error)
	assert.Equal(t, int64(1), dispatches)
	assert.Equal(t, int64(1), executions)
	assert.Equal(t, int64(1), events)
}

func Test__DispatchWorkOrder__ConcurrentRetriesCreateOneDispatch(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	key := "concurrent-dispatch"
	request := &pb.DispatchWorkOrderRequest{
		FactoryId:      factoryModel.ID.String(),
		OrderId:        order.ID.String(),
		LineName:       line.Name,
		IdempotencyKey: &key,
	}

	type result struct {
		response *pb.DispatchWorkOrderResponse
		err      error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	var workers sync.WaitGroup
	workers.Add(2)
	for range 2 {
		go func() {
			defer workers.Done()
			<-start
			response, dispatchErr := DispatchWorkOrder(ctx, r.Organization.ID.String(), request)
			results <- result{response: response, err: dispatchErr}
		}()
	}
	close(start)
	workers.Wait()
	close(results)

	var dispatchID string
	for result := range results {
		require.NoError(t, result.err)
		require.Len(t, result.response.Order.LineDispatches, 1)
		if dispatchID == "" {
			dispatchID = result.response.Order.LineDispatches[0].Id
		}
		assert.Equal(t, dispatchID, result.response.Order.LineDispatches[0].Id)
	}

	var dispatches, executions int64
	require.NoError(t, db.Model(&models.FactoryWorkOrderLineDispatch{}).Where("work_order_id = ?", order.ID).Count(&dispatches).Error)
	require.NoError(t, db.Model(&models.FactoryWorkOrderExecution{}).Where("work_order_id = ?", order.ID).Count(&executions).Error)
	assert.Equal(t, int64(1), dispatches)
	assert.Equal(t, int64(1), executions)
}

func Test__DispatchWorkOrder__ConcurrentDifferentKeysCreateOneDispatch(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	start := make(chan struct{})
	errors := make(chan error, 2)
	var workers sync.WaitGroup
	workers.Add(2)
	for _, key := range []string{"concurrent-dispatch-1", "concurrent-dispatch-2"} {
		go func() {
			defer workers.Done()
			<-start
			_, dispatchErr := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
				FactoryId: factoryModel.ID.String(), OrderId: order.ID.String(), LineName: line.Name, IdempotencyKey: &key,
			})
			errors <- dispatchErr
		}()
	}
	close(start)
	workers.Wait()
	close(errors)

	var successes, conflicts int
	for dispatchErr := range errors {
		if dispatchErr == nil {
			successes++
			continue
		}
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(dispatchErr))
		conflicts++
	}
	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, conflicts)

	var requests, dispatches, executions int64
	require.NoError(t, db.Model(&models.FactoryWorkOrderDispatchRequest{}).Where("work_order_id = ?", order.ID).Count(&requests).Error)
	require.NoError(t, db.Model(&models.FactoryWorkOrderLineDispatch{}).Where("work_order_id = ?", order.ID).Count(&dispatches).Error)
	require.NoError(t, db.Model(&models.FactoryWorkOrderExecution{}).Where("work_order_id = ?", order.ID).Count(&executions).Error)
	assert.Equal(t, int64(1), requests)
	assert.Equal(t, int64(1), dispatches)
	assert.Equal(t, int64(1), executions)
}

func Test__DispatchWorkOrder__RejectsKeyReuseWithDifferentParameters(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	key := "conflicting-dispatch"
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(), OrderId: order.ID.String(), LineName: line.Name, IdempotencyKey: &key,
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(), OrderId: order.ID.String(), LineName: "other-line", IdempotencyKey: &key,
	})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
}

func Test__DispatchWorkOrder__FailedAttemptDoesNotConsumeKey(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	key := "recoverable-dispatch"

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(), OrderId: order.ID.String(), LineName: "ship", IdempotencyKey: &key,
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, grpcerrors.Code(err))

	var requests int64
	require.NoError(t, db.Model(&models.FactoryWorkOrderDispatchRequest{}).Where("work_order_id = ?", order.ID).Count(&requests).Error)
	assert.Zero(t, requests)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(), OrderId: order.ID.String(), LineName: line.Name, IdempotencyKey: &key,
	})
	require.NoError(t, err)
	require.Len(t, response.Order.LineDispatches, 1)
}

func Test__DispatchWorkOrder__LateFailureDoesNotConsumeKey(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	first, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
	})
	require.NoError(t, err)
	require.Len(t, first.Order.LineDispatches, 1)

	key := "retry-after-active-conflict"
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(), OrderId: order.ID.String(), LineName: line.Name, IdempotencyKey: &key,
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))

	var requests int64
	require.NoError(t, db.Model(&models.FactoryWorkOrderDispatchRequest{}).Where("work_order_id = ?", order.ID).Count(&requests).Error)
	assert.Zero(t, requests)

	active, err := models.FindWorkOrderLineDispatch(db, uuid.MustParse(first.Order.LineDispatches[0].Id))
	require.NoError(t, err)
	require.NoError(t, active.Finish(db, models.CanvasRunResultFailed))

	response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(), OrderId: order.ID.String(), LineName: line.Name, IdempotencyKey: &key,
	})
	require.NoError(t, err)
	require.Len(t, response.Order.LineDispatches, 2)
}

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
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
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
	assert.Empty(t, dispatch.Model)
	require.Len(t, dispatch.Steps, 1)
	assert.Equal(t, app.Name, dispatch.Steps[0].Name)
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
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
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
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
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

func Test__DispatchWorkOrder__ReplaceActiveFinishesCurrentAndStartsAgain(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(database.DB(t.Context()), "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	firstResp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	firstID := firstResp.Order.LineDispatches[0].Id

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId:     factoryModel.ID.String(),
		OrderId:       order.ID.String(),
		LineName:      line.Name,
		ReplaceActive: true,
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.LineDispatches, 2)
	assert.Equal(t, firstID, resp.Order.LineDispatches[0].Id)
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_FINISHED, resp.Order.LineDispatches[0].State)
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_ACTIVE, resp.Order.LineDispatches[1].State)
	assert.Equal(t, int32(0), resp.Order.LineDispatches[1].StepExecutions[0].StepIndex)
}

func Test__DispatchWorkOrder__StartsAtRequestedStep(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-two", "start-two")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	})
	require.NoError(t, err)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId:      factoryModel.ID.String(),
		OrderId:        order.ID.String(),
		LineName:       line.Name,
		StartStepIndex: 1,
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.LineDispatches, 1)
	require.Len(t, resp.Order.LineDispatches[0].StepExecutions, 1)
	assert.Equal(t, int32(1), resp.Order.LineDispatches[0].StepExecutions[0].StepIndex)
	assert.Equal(t, secondApp.Name, resp.Order.LineDispatches[0].StepExecutions[0].Step)
}

func Test__DispatchWorkOrder__RerunStepKeepsEarlierExecutions(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Improve AGENTS.md", "", &r.User, nil, nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "plan", "start-plan")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-implement")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	})
	require.NoError(t, err)

	firstResp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	require.Len(t, firstResp.Order.LineDispatches, 1)
	firstID := firstResp.Order.LineDispatches[0].Id

	firstDispatch, err := models.FindWorkOrderLineDispatch(db, uuid.MustParse(firstID))
	require.NoError(t, err)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, startErr := firstDispatch.EnqueueOrStartStep(tx, order, 1)
		if startErr != nil {
			return startErr
		}
		return firstDispatch.Finish(tx, models.CanvasRunResultFailed)
	}))

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId:      factoryModel.ID.String(),
		OrderId:        order.ID.String(),
		LineName:       line.Name,
		StartStepIndex: 1,
		ReplaceActive:  true,
	})
	require.NoError(t, err)

	var planCount int
	for _, dispatch := range resp.Order.LineDispatches {
		if dispatch.Id == firstID {
			for _, execution := range dispatch.StepExecutions {
				if execution.StepIndex == 0 {
					planCount++
				}
			}
		}
	}
	assert.Equal(t, 1, planCount, "rerun of Implementation must keep the Planning execution")
	require.Len(t, resp.Order.LineDispatches, 1)
	require.GreaterOrEqual(t, len(resp.Order.LineDispatches[0].StepExecutions), 2)
}

func Test__DispatchWorkOrder__ReturnsOwnerNameAfterStart(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", nil, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Assignees, 1)
	assert.Equal(t, r.User.String(), resp.Order.Assignees[0].Id)
	assert.Equal(t, r.UserModel.Name, resp.Order.Assignees[0].Name)
	assert.NotEqual(t, r.User.String(), resp.Order.Assignees[0].Name)
}

func Test__DispatchWorkOrder__PersistsChosenModel(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-4-6", "claude-opus-4-6")

	app := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "hosted", "claude-sonnet-4-6")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
		Model:     "claude-opus-4-6",
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.LineDispatches, 1)
	assert.Equal(t, "claude-opus-4-6", resp.Order.LineDispatches[0].Model)

	active, err := order.FindActiveLineDispatch(db)
	require.NoError(t, err)
	assert.Equal(t, "claude-opus-4-6", active.Model)
}

func Test__DispatchWorkOrder__RejectsModelNotOnLine(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-4-6")
	seedHostedModels(t, db, models.UsageProviderOpenAI, "gpt-5")

	app := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "hosted", "claude-sonnet-4-6")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
		Model:     "gpt-5",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}
