package workers

import (
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

func Test__BacklogAdmission_ConcurrentEventsShareOrganizationCapacity(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(models.BillingEnabledEnv, "yes")
	setTaskPlanLimit(t, r, 1)
	var events []*models.CanvasEvent
	for range 3 {
		event, _ := createBacklogAdmissionEvent(t, r)
		events = append(events, event)
	}
	router := NewEventRouter("")
	logger := log.NewEntry(log.New())
	var wg sync.WaitGroup
	failures := make(chan error, len(events))
	for _, event := range events {
		wg.Go(func() { failures <- router.LockAndProcessEvent(logger, *event, time.Now()) })
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}

	db := database.DB(t.Context())
	var work []models.CanvasNodeQueueItem
	require.NoError(t, db.Find(&work).Error)
	require.Len(t, work, 1, "only one Backlog agent across workspaces may receive executable work")
	var pending []models.CanvasEvent
	require.NoError(t, db.Where("state = ?", models.CanvasEventStatePending).Order("created_at, id").Find(&pending).Error)
	require.Len(t, pending, 2)
	for _, event := range pending {
		var queued int64
		require.NoError(t, db.Model(&models.CanvasNodeQueueItem{}).Where("event_id = ?", event.ID).Count(&queued).Error)
		assert.Zero(t, queued, "waiting events must not dispatch agent work")
	}

	// Repeated delivery cannot create more runs while the first is active.
	require.NoError(t, router.LockAndProcessEvent(logger, pending[0], time.Now()))
	run, err := models.FindUnscopedCanvasRun(db, work[0].RunID)
	require.NoError(t, err)
	finishRun(t, run, models.CanvasRunResultPassed)
	require.NoError(t, router.LockAndProcessEvent(logger, pending[0], time.Now()))
	require.NoError(t, router.LockAndProcessEvent(logger, pending[1], time.Now()))
	var active int64
	require.NoError(t, db.Table("workflow_node_queue_items AS q").Joins("JOIN workflow_runs r ON r.id = q.run_id").
		Where("r.state = ?", models.CanvasRunStateStarted).Count(&active).Error)
	assert.Equal(t, int64(1), active, "completion must admit exactly one waiting Backlog run")
}

func Test__BacklogAdmission_SharesCapacityWithLineTasks(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(models.BillingEnabledEnv, "yes")
	setTaskPlanLimit(t, r, 1)
	event, _ := createBacklogAdmissionEvent(t, r)
	router := NewEventRouter("")
	logger := log.NewEntry(log.New())
	require.NoError(t, router.LockAndProcessEvent(logger, *event, time.Now()))
	backlogRun, err := models.FindCanvasRunByRootEventInTransaction(database.DB(t.Context()), event.ID)
	require.NoError(t, err)

	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
	_, result := fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Line task"))
	require.NotNil(t, result.QueueItem, "an active Backlog run must hold the organization slot")
	session, err := models.FindPlanningSessionByRun(database.DB(t.Context()), backlogRun.ID)
	require.NoError(t, err)
	require.NoError(t, session.End(database.DB(t.Context())))
	require.NoError(t, database.DB(t.Context()).Model(backlogRun).Update("state", models.CanvasRunStateCancelling).Error)
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
	assertOrganizationTaskCounts(t, r.Organization.ID, 0, 1)
	finishRun(t, backlogRun, models.CanvasRunResultPassed)
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
	assertOrganizationTaskCounts(t, r.Organization.ID, 1, 0)

	waiting, _ := createBacklogAdmissionEvent(t, r)
	require.NoError(t, router.LockAndProcessEvent(logger, *waiting, time.Now()))
	updated, err := models.FindCanvasEvent(waiting.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasEventStatePending, updated.State, "a reserved line task must block Backlog admission")
	var queued int64
	require.NoError(t, database.DB(t.Context()).Model(&models.CanvasNodeQueueItem{}).Where("event_id = ?", waiting.ID).Count(&queued).Error)
	assert.Zero(t, queued)
}

func createBacklogAdmissionEvent(t *testing.T, r *support.ResourceRegistry) (*models.CanvasEvent, *models.FactoryWorkOrder) {
	t.Helper()
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("backlog"), "", "")
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: models.FactoryAppBacklogTriggerID, Type: models.NodeTypeTrigger,
			Ref:      datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onWorkOrder"}}),
			Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateBacklogID, 4)),
		},
		{NodeID: "refine-task", Type: models.NodeTypeComponent},
	}, []models.Edge{{SourceID: models.FactoryAppBacklogTriggerID, TargetID: "refine-task", Channel: "default"}})
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	order, err := factory.CreateWorkOrder(db, "Refine task", "", &r.User, nil, nil)
	require.NoError(t, err)
	return support.EmitCanvasEventForNodeWithData(t, canvas.ID, models.FactoryAppBacklogTriggerID, "default", nil, map[string]any{
		"type": "workOrder.created",
		"data": map[string]any{
			models.WorkOrderCreatedRefinementEnabledDataKey: true,
			"workOrder": map[string]any{"id": order.ID.String()},
		},
	}), order
}

func Test__BacklogAdmission_QueuePositionAndCancellation(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(models.BillingEnabledEnv, "yes")
	setTaskPlanLimit(t, r, 1)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	db := database.DB(ctx)
	router := NewEventRouter("")
	logger := log.NewEntry(log.New())
	active, _ := createBacklogAdmissionEvent(t, r)
	require.NoError(t, router.LockAndProcessEvent(logger, *active, time.Now()))
	first, firstOrder := createBacklogAdmissionEvent(t, r)
	second, secondOrder := createBacklogAdmissionEvent(t, r)
	for _, event := range []*models.CanvasEvent{first, second} {
		require.NoError(t, router.LockAndProcessEvent(logger, *event, time.Now()))
	}
	page, err := factoryactions.ListWorkOrders(ctx, r.Organization.ID.String(), &pb.ListWorkOrdersRequest{FactoryId: secondOrder.FactoryID.String()})
	require.NoError(t, err)
	require.Len(t, page.Orders, 1)
	require.NotNil(t, page.Orders[0].QueuedTasks)
	assert.Equal(t, int32(2), page.Orders[0].QueuedTasks[0].Position, "position includes queued work in other workspaces")

	request := &pb.CancelWorkOrderQueueRequest{FactoryId: firstOrder.FactoryID.String(), OrderId: firstOrder.ID.String(), QueueId: second.ID.String()}
	_, err = factoryactions.CancelWorkOrderQueue(ctx, r.Organization.ID.String(), request)
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err), "another task's pending work cannot be cancelled")
	request.QueueId = first.ID.String()
	response, err := factoryactions.CancelWorkOrderQueue(ctx, r.Organization.ID.String(), request)
	require.NoError(t, err)
	assert.Equal(t, pb.WorkOrder_STATE_DRAFT, response.Order.State)
	assert.Nil(t, response.Order.QueuedTasks)
	_, err = factoryactions.CancelWorkOrderQueue(ctx, r.Organization.ID.String(), request)
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	require.NoError(t, router.LockAndProcessEvent(logger, *first, time.Now()))
	var firstWork int64
	require.NoError(t, db.Model(&models.CanvasNodeQueueItem{}).Where("event_id = ?", first.ID).Count(&firstWork).Error)
	assert.Zero(t, firstWork, "cancelled queued events must not dispatch on a later sweep")

	page, err = factoryactions.ListWorkOrders(ctx, r.Organization.ID.String(), &pb.ListWorkOrdersRequest{FactoryId: secondOrder.FactoryID.String()})
	require.NoError(t, err)
	assert.Equal(t, int32(1), page.Orders[0].QueuedTasks[0].Position)
	run, err := models.FindUnscopedCanvasRun(db, active.RunID)
	require.NoError(t, err)
	finishRun(t, run, models.CanvasRunResultPassed)
	require.NoError(t, router.LockAndProcessEvent(logger, *second, time.Now()))
	_, err = factoryactions.CancelWorkOrderQueue(ctx, r.Organization.ID.String(), &pb.CancelWorkOrderQueueRequest{FactoryId: secondOrder.FactoryID.String(), OrderId: secondOrder.ID.String(), QueueId: second.ID.String()})
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err), "a stale queue button must not cancel an admitted agent")
	run, err = models.FindUnscopedCanvasRun(db, second.RunID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, run.State)
}

func Test__BacklogAdmission_UnscheduleKeepsTaskInBacklogAndAdvancesQueue(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(models.BillingEnabledEnv, "yes")
	setTaskPlanLimit(t, r, 1)
	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
	_, active := fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Active"))
	db := database.DB(t.Context())
	order, err := fixture.factory.CreateWorkOrder(db, "Waiting", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, order, err = factoryactions.DispatchWorkOrderOnLine(db, fixture.factory, order.ID, fixture.line, &r.User, 0, false, "", "")
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateDraft, order.State, "a pending start must stay in Backlog")
	dispatch, err := order.FindActiveLineDispatch(db)
	require.NoError(t, err)
	queue := queueItemForDispatch(t, dispatch.ID)
	require.NotNil(t, queue)
	next := fixture.createOpenWorkOrder(t, r, "Next")
	nextDispatch, _ := fixture.dispatchLine(t, next)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	response, err := factoryactions.CancelWorkOrderQueue(ctx, r.Organization.ID.String(), &pb.CancelWorkOrderQueueRequest{FactoryId: fixture.factory.ID.String(), OrderId: order.ID.String(), QueueId: queue.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, pb.WorkOrder_STATE_DRAFT, response.Order.State)
	assert.Nil(t, response.Order.QueuedTasks)
	assert.Equal(t, models.CanvasRunResultCancelled, reloadDispatch(t, dispatch.ID).Result)
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
	assert.Empty(t, executionsForOrder(t, order.ID))
	assert.Equal(t, 1, queueItemForDispatch(t, nextDispatch.ID).Position)
	finishRun(t, active.Run, models.CanvasRunResultPassed)
	advanceFactoryLine(t, r, active.Run.ID)
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
	assert.Len(t, executionsForOrder(t, next.ID), 1)
	assert.Empty(t, executionsForOrder(t, order.ID), "unscheduling must not start the task later")
	_, _, err = factoryactions.DispatchWorkOrderOnLine(database.DB(ctx), fixture.factory, order.ID, fixture.line, &r.User, 0, false, "", "")
	require.NoError(t, err, "the task must remain available to start again")
	nextExecution := executionsForOrder(t, next.ID)[0]
	nextRun, err := models.FindUnscopedCanvasRun(db, *nextExecution.RunID)
	require.NoError(t, err)
	finishRun(t, nextRun, models.CanvasRunResultPassed)
	advanceFactoryLine(t, r, nextRun.ID)
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
	order, err = fixture.factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, order.State, "admission must move the task out of Backlog")
	assert.Len(t, executionsForOrder(t, order.ID), 1)
}

func Test__BacklogAdmission_QueuePositionsMatchAdmissionOrder(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(models.BillingEnabledEnv, "yes")
	setTaskPlanLimit(t, r, 1)
	older, _ := createBacklogAdmissionEvent(t, r)
	fixture := setupStepQueueLine(t, r, []*int{stepMaxParallelism(10)})
	_, waiting := fixture.dispatchLine(t, fixture.createOpenWorkOrder(t, r, "Newer line task"))
	require.NotNil(t, waiting.QueueItem, "the line task must join behind the older Backlog event")
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
	assertOrganizationTaskCounts(t, r.Organization.ID, 0, 1)
	router := NewEventRouter("")
	logger := log.NewEntry(log.New())
	require.NoError(t, router.LockAndProcessEvent(logger, *older, time.Now()))
	newer, _ := createBacklogAdmissionEvent(t, r)
	run, err := models.FindUnscopedCanvasRun(database.DB(t.Context()), older.RunID)
	require.NoError(t, err)
	finishRun(t, run, models.CanvasRunResultPassed)
	require.NoError(t, router.LockAndProcessEvent(logger, *newer, time.Now()))
	var work int64
	require.NoError(t, database.DB(t.Context()).Model(&models.CanvasNodeQueueItem{}).Where("event_id = ?", newer.ID).Count(&work).Error)
	assert.Zero(t, work, "the newer Backlog event must not overtake the waiting line task")
	NewRunFinalizer("", r.Registry).sweepQueuedTasks(t.Context())
	assertOrganizationTaskCounts(t, r.Organization.ID, 1, 0)
}

func Test__TaskAdmission_RootAutomationsShareSlotAndCancelIndependently(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(models.BillingEnabledEnv, "yes")
	setTaskPlanLimit(t, r, 1)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	db := database.DB(ctx)
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("automations"), "", "")
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "start", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}})},
		{NodeID: "work", Type: models.NodeTypeComponent},
	}, []models.Edge{{SourceID: "start", TargetID: "work", Channel: "default"}})
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	order, err := factory.CreateWorkOrder(db, "Parallel automations", "", &r.User, nil, nil)
	require.NoError(t, err)
	emit := func(wrapped bool) *models.CanvasEvent {
		payload := map[string]any{"workOrder": map[string]any{"id": order.ID.String()}}
		if wrapped {
			payload = map[string]any{"data": payload}
		}
		return support.EmitCanvasEventForNodeWithData(t, canvas.ID, "start", "default", nil, payload)
	}
	first, second := emit(false), emit(true)
	queued, err := models.ListOrganizationQueuedTasks(db, r.Organization.ID)
	require.NoError(t, err)
	require.Len(t, queued, 2, "both pending automations must be visible")
	assert.Equal(t, queued[0].Position, queued[1].Position, "one task has one queue position")
	router := NewEventRouter("")
	logger := log.NewEntry(log.New())
	for _, event := range []*models.CanvasEvent{first, second} {
		require.NoError(t, router.LockAndProcessEvent(logger, *event, time.Now()))
		var work int64
		require.NoError(t, db.Model(&models.CanvasNodeQueueItem{}).Where("event_id = ?", event.ID).Count(&work).Error)
		assert.Equal(t, int64(1), work, "automations for the active task share its slot")
	}
	pending := emit(true)
	response, err := factoryactions.CancelWorkOrderQueue(ctx, r.Organization.ID.String(), &pb.CancelWorkOrderQueueRequest{
		FactoryId: factory.ID.String(), OrderId: order.ID.String(), QueueId: pending.ID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, pb.WorkOrder_STATE_DRAFT, response.Order.State)
	assert.Empty(t, response.Order.QueuedTasks)
	waiter, _ := createBacklogAdmissionEvent(t, r)
	for _, event := range []*models.CanvasEvent{second, first} {
		require.NoError(t, router.LockAndProcessEvent(logger, *waiter, time.Now()))
		updated, err := models.FindCanvasEvent(waiter.ID)
		require.NoError(t, err)
		assert.Equal(t, models.CanvasEventStatePending, updated.State, "the remaining automation must hold capacity")
		run, err := models.FindUnscopedCanvasRun(db, event.RunID)
		require.NoError(t, err)
		assert.Equal(t, models.CanvasRunStateStarted, run.State, "unscheduling another automation must not cancel this run")
		finishRun(t, run, models.CanvasRunResultPassed)
	}
	require.NoError(t, router.LockAndProcessEvent(logger, *waiter, time.Now()))
	updated, err := models.FindCanvasEvent(waiter.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasEventStateRouted, updated.State)
}
