package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/protobuf/proto"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__RunFinalizer_FinalizesRunAfterTerminalExecutionEvent(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	router := NewEventRouter(amqpURL)
	finalizer := NewRunFinalizer(amqpURL, r.Registry)
	logger := log.NewEntry(log.New())

	trigger := "trigger-1"
	node := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: trigger, Type: models.NodeTypeTrigger},
			{NodeID: node, Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: trigger, TargetID: node, Channel: "default"},
		},
	)

	triggerEvent := support.EmitCanvasEventForNode(t, canvas.ID, trigger, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), triggerEvent)
	require.NoError(t, err)
	require.NoError(t, triggerEvent.Routed())

	execution := support.CreateCanvasNodeExecution(t, canvas.ID, node, triggerEvent.ID, triggerEvent.ID)
	execution.RunID = run.ID
	require.NoError(t, database.Conn().Save(execution).Error)
	_, err = execution.Pass(map[string][]any{"default": {map[string]any{}}})
	require.NoError(t, err)

	events, err := models.ListCanvasEvents(database.Conn(), canvas.ID, node, 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)

	err = router.LockAndProcessEvent(logger, events[0], time.Now())
	require.NoError(t, err)

	updatedRun, err := models.FindCanvasRunByRootEventInTransaction(database.Conn(), triggerEvent.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, updatedRun.State)

	require.NoError(t, finalizer.finalizeRun(canvas.ID, run.ID, runFinalizerTriggerEventTerminal))

	updatedRun, err = models.FindCanvasRunByRootEventInTransaction(database.Conn(), triggerEvent.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultPassed, updatedRun.Result)
	assert.NotNil(t, updatedRun.FinishedAt)
}

func Test__RunFinalizer_FinalizesRunAfterQueueItemDeleted(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)

	node := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: node, Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, node, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), event)
	require.NoError(t, err)
	require.NoError(t, event.Routed())

	execution := support.CreateCanvasNodeExecution(t, canvas.ID, node, event.ID, event.ID)
	execution.RunID = run.ID
	require.NoError(t, database.Conn().Save(execution).Error)
	require.NoError(t, execution.Cancel(nil))

	body, err := proto.Marshal(&pb.CanvasNodeQueueItemMessage{
		Id:       event.ID.String(),
		CanvasId: canvas.ID.String(),
		NodeId:   node,
		RunId:    run.ID.String(),
	})
	require.NoError(t, err)

	require.NoError(t, finalizer.consumeQueueItemDeleted(tackle.NewFakeDelivery(body)))

	updatedRun, err := models.FindCanvasRunByRootEventInTransaction(database.Conn(), event.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultCancelled, updatedRun.Result)
	assert.NotNil(t, updatedRun.FinishedAt)
}

func Test__RunFinalizer_DoesNotFinalizeRunWithOpenWork(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)

	trigger := "trigger-1"
	node := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: trigger, Type: models.NodeTypeTrigger},
			{NodeID: node, Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: trigger, TargetID: node, Channel: "default"},
		},
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, trigger, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), event)
	require.NoError(t, err)

	now := time.Now()
	queueItem := models.CanvasNodeQueueItem{
		WorkflowID:  canvas.ID,
		NodeID:      node,
		RootEventID: event.ID,
		RunID:       run.ID,
		EventID:     event.ID,
		CreatedAt:   &now,
	}
	require.NoError(t, database.Conn().Create(&queueItem).Error)

	require.NoError(t, finalizer.finalizeRun(canvas.ID, run.ID, runFinalizerTriggerEventTerminal))

	updatedRun, err := models.FindCanvasRunByRootEventInTransaction(database.Conn(), event.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, updatedRun.State)
}

func Test__RunFinalizer_SweepTouchesUpdatedAtWhenRunHasOpenWork(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)

	trigger := "trigger-1"
	node := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: trigger, Type: models.NodeTypeTrigger},
			{NodeID: node, Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: trigger, TargetID: node, Channel: "default"},
		},
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, trigger, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), event)
	require.NoError(t, err)

	staleUpdatedAt := time.Now().Add(-time.Hour)
	require.NoError(t, database.Conn().Model(run).Update("updated_at", &staleUpdatedAt).Error)

	now := time.Now()
	queueItem := models.CanvasNodeQueueItem{
		WorkflowID:  canvas.ID,
		NodeID:      node,
		RootEventID: event.ID,
		RunID:       run.ID,
		EventID:     event.ID,
		CreatedAt:   &now,
	}
	require.NoError(t, database.Conn().Create(&queueItem).Error)

	require.NoError(t, finalizer.finalizeRun(canvas.ID, run.ID, runFinalizerTriggerEventTerminal))

	unchangedRun, err := models.FindCanvasRunByRootEventInTransaction(database.Conn(), event.ID)
	require.NoError(t, err)
	assert.Equal(t, staleUpdatedAt.Unix(), unchangedRun.UpdatedAt.Unix())

	require.NoError(t, finalizer.finalizeRun(canvas.ID, run.ID, runFinalizerTriggerSweep))

	touchedRun, err := models.FindCanvasRunByRootEventInTransaction(database.Conn(), event.ID)
	require.NoError(t, err)
	assert.True(t, touchedRun.UpdatedAt.After(staleUpdatedAt))
	assert.Equal(t, models.CanvasRunStateStarted, touchedRun.State)
}

func Test__RunFinalizer_FinalizesCancellingRunWithForcedCancelledResult(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)

	node := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: node, Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, node, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), event)
	require.NoError(t, err)
	require.NoError(t, event.Routed())

	execution := support.CreateCanvasNodeExecution(t, canvas.ID, node, event.ID, event.ID)
	execution.RunID = run.ID
	require.NoError(t, database.Conn().Save(execution).Error)
	require.NoError(t, execution.Cancel(nil))

	now := time.Now()
	require.NoError(t, database.Conn().Model(run).Updates(map[string]any{
		"state":        models.CanvasRunStateCancelling,
		"cancelled_at": now,
		"cancelled_by": r.User,
	}).Error)

	require.NoError(t, finalizer.finalizeRun(canvas.ID, run.ID, runFinalizerTriggerExecutionFinished))

	updatedRun, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultCancelled, updatedRun.Result)
}

func Test__RunFinalizer_SweepCancellingRuns_FinalizesWhenNoOpenWork(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)

	node := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: node, Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, node, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), event)
	require.NoError(t, err)
	require.NoError(t, event.Routed())

	execution := support.CreateCanvasNodeExecution(t, canvas.ID, node, event.ID, event.ID)
	execution.RunID = run.ID
	require.NoError(t, database.Conn().Save(execution).Error)
	require.NoError(t, execution.Cancel(nil))

	now := time.Now()
	require.NoError(t, database.Conn().Model(run).Updates(map[string]any{
		"state":        models.CanvasRunStateCancelling,
		"cancelled_at": now,
		"cancelled_by": r.User,
	}).Error)

	require.NoError(t, finalizer.sweepCancellingRuns())

	updatedRun, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultCancelled, updatedRun.Result)
	assert.NotNil(t, updatedRun.FinishedAt)
}

func Test__RunFinalizer_SweepCancellingRuns_DrainsOpenWorkAndPublishesMessages(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)

	executionCancellingConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionCancellingRoutingKey)
	executionCancellingConsumer.Start()
	defer executionCancellingConsumer.Stop()

	queueItemDeletedConsumer := testconsumer.New(amqpURL, messages.CanvasQueueItemDeletedRoutingKey)
	queueItemDeletedConsumer.Start()
	defer queueItemDeletedConsumer.Stop()

	node := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: node, Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, node, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), event)
	require.NoError(t, err)
	require.NoError(t, event.Routed())

	execution := support.CreateCanvasNodeExecution(t, canvas.ID, node, event.ID, event.ID)
	execution.RunID = run.ID
	execution.State = models.CanvasNodeExecutionStateStarted
	require.NoError(t, database.Conn().Save(execution).Error)

	now := time.Now()
	queueItem := models.CanvasNodeQueueItem{
		WorkflowID:  canvas.ID,
		NodeID:      node,
		RootEventID: event.ID,
		RunID:       run.ID,
		EventID:     event.ID,
		CreatedAt:   &now,
	}
	require.NoError(t, database.Conn().Create(&queueItem).Error)

	require.NoError(t, database.Conn().Model(run).Updates(map[string]any{
		"state":        models.CanvasRunStateCancelling,
		"cancelled_at": now,
		"cancelled_by": r.User,
	}).Error)

	require.NoError(t, finalizer.sweepCancellingRuns())

	assert.True(t, executionCancellingConsumer.HasReceivedMessage())
	assert.True(t, queueItemDeletedConsumer.HasReceivedMessage())

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)

	var queueItemCount int64
	require.NoError(t, database.Conn().Model(&models.CanvasNodeQueueItem{}).Where("run_id = ?", run.ID).Count(&queueItemCount).Error)
	assert.Zero(t, queueItemCount)

	updatedRun, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, updatedRun.State)

	require.NoError(t, execution.Cancel(nil))

	require.NoError(t, finalizer.sweepCancellingRuns())

	updatedRun, err = models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultCancelled, updatedRun.Result)
	assert.NotNil(t, updatedRun.FinishedAt)
}

func Test__RunFinalizer_SweepCancellingRuns_DoesNotBumpUpdatedAtWhenStillOpenAfterDrain(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)

	node := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: node, Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, node, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), event)
	require.NoError(t, err)
	require.NoError(t, event.Routed())

	execution := support.CreateCanvasNodeExecution(t, canvas.ID, node, event.ID, event.ID)
	execution.RunID = run.ID
	execution.State = models.CanvasNodeExecutionStateStarted
	require.NoError(t, database.Conn().Save(execution).Error)

	staleUpdatedAt := time.Now().Add(-time.Hour)
	now := time.Now()
	require.NoError(t, database.Conn().Model(run).Updates(map[string]any{
		"state":        models.CanvasRunStateCancelling,
		"cancelled_at": now,
		"cancelled_by": r.User,
		"updated_at":   staleUpdatedAt,
	}).Error)

	require.NoError(t, finalizer.sweepCancellingRuns())

	unchangedRun, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, staleUpdatedAt.Unix(), unchangedRun.UpdatedAt.Unix())
	assert.Equal(t, models.CanvasRunStateCancelling, unchangedRun.State)
}

func Test__RunFinalizer__CallsFinishedCallbackWhenConfigured(t *testing.T) {
	actionName := "run_finalizer_finished_cb_" + uuid.New().String()
	hookCalled := false
	var receivedCallback core.RunFinishedCallback

	registry.RegisterAction(actionName, impl.NewDummyAction(impl.DummyActionOptions{
		Name:  actionName,
		Hooks: []core.Hook{{Name: "onRunFinished", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			hookCalled = true
			var err error
			receivedCallback, err = core.DecodeRunFinishedCallback(ctx.Parameters)
			return err
		},
	}))

	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)

	parentCanvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "runApp",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: actionName}}),
			},
		},
		[]models.Edge{
			{SourceID: "trigger", TargetID: "runApp", Channel: "default"},
		},
	)
	childCanvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "entry", Type: models.NodeTypeTrigger},
			{NodeID: "component", Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: "entry", TargetID: "component", Channel: "default"},
		},
	)

	parentRootEvent := support.EmitCanvasEventForNode(t, parentCanvas.ID, "trigger", "default", nil)
	parentRun, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), parentRootEvent)
	require.NoError(t, err)
	require.NoError(t, parentRootEvent.Routed())

	parentExecution := support.CreateCanvasNodeExecution(t, parentCanvas.ID, "runApp", parentRootEvent.ID, parentRootEvent.ID)
	parentExecution.RunID = parentRun.ID
	require.NoError(t, database.Conn().Save(parentExecution).Error)

	childRun := createSubRunWithCallbacks(t, subRunOptions{
		workflowID:        childCanvas.ID,
		nodeID:            "entry",
		parentRunID:       parentRun.ID,
		parentWorkflowID:  parentCanvas.ID,
		parentExecutionID: parentExecution.ID,
		state:             models.CanvasRunStateStarted,
		callbacks: []core.RunCallback{
			{
				When: core.RunCallbackWhenFinished,
				On:   core.RunCallbackOnParent,
				Hook: "onRunFinished",
			},
		},
	})

	childRootEvent := support.EmitCanvasEventForNode(t, childCanvas.ID, "entry", "default", nil)
	require.NoError(t, database.Conn().Model(&childRootEvent).Update("run_id", childRun.ID).Error)
	require.NoError(t, childRootEvent.Routed())

	childExecution := support.CreateCanvasNodeExecution(t, childCanvas.ID, "component", childRootEvent.ID, childRootEvent.ID)
	childExecution.RunID = childRun.ID
	require.NoError(t, database.Conn().Save(childExecution).Error)
	_, err = childExecution.Pass(map[string][]any{"default": {map[string]any{}}})
	require.NoError(t, err)

	events, err := models.ListCanvasEvents(database.Conn(), childCanvas.ID, "component", 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)

	router := NewEventRouter(amqpURL)
	logger := log.NewEntry(log.New())
	require.NoError(t, router.LockAndProcessEvent(logger, events[0], time.Now()))

	require.NoError(t, finalizer.finalizeRun(childCanvas.ID, childRun.ID, runFinalizerTriggerEventTerminal))

	updatedChildRun, err := models.FindCanvasRunInTransaction(database.Conn(), childCanvas.ID, childRun.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateFinished, updatedChildRun.State)
	assert.Equal(t, models.CanvasRunResultPassed, updatedChildRun.Result)

	assert.True(t, hookCalled)
	assert.Equal(t, childRun.ID, receivedCallback.Run.ID)
	assert.Equal(t, childCanvas.ID, receivedCallback.Run.AppID)
	assert.Equal(t, models.CanvasRunResultPassed, receivedCallback.Run.Result)
	assert.Equal(t, "runApp", parentExecution.NodeID)
}

func Test__RunFinalizer__ExecuteNextFactoryLineStep(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(database.Conn(), "Ship feature", "", &r.User, nil, nil)
	require.NoError(t, err)
	dispatchWorkOrderForTest(t, order)

	line, err := factory.CreateLine(database.Conn(), "ship", nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-two", "start-two")

	steps := []models.FactoryLineStep{
		{
			Type:       models.FactoryLineStepTypeRunApp,
			AppID:      firstApp.ID,
			Entrypoint: firstEntry,
		},
		{
			Type:       models.FactoryLineStepTypeRunApp,
			AppID:      secondApp.ID,
			Entrypoint: secondEntry,
		},
	}
	require.NoError(t, line.Update(database.Conn(), nil, steps))

	var firstResult *models.FactoryLineStepResult
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var startErr error
		_, firstResult, startErr = line.Dispatch(tx, order)
		return startErr
	}))

	now := time.Now()
	require.NoError(t, database.Conn().Model(firstResult.Run).Updates(map[string]any{
		"state":       models.CanvasRunStateFinished,
		"result":      models.CanvasRunResultPassed,
		"updated_at":  &now,
		"finished_at": &now,
	}).Error)

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)
	var pending []factoryLinePendingRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var advanceErr error
		pending, advanceErr = finalizer.executeNextFactoryLineStep(tx, firstResult.Run.ID)
		return advanceErr
	}))

	require.Len(t, pending, 1)
	assert.Equal(t, secondApp.ID, pending[0].workflowID)

	firstExecution, err := models.FindWorkOrderExecutionByRunID(database.Conn(), firstResult.Run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusFinished, firstExecution.Status)
	assert.Equal(t, models.CanvasRunResultPassed, firstExecution.Result)

	secondExecution, err := models.FindWorkOrderExecutionByRunID(database.Conn(), pending[0].runID)
	require.NoError(t, err)
	assert.Equal(t, 1, secondExecution.StepIndex)
	assert.Equal(t, secondApp.Name, secondExecution.StepName)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, secondExecution.Status)
	assert.Equal(t, firstExecution.LineDispatchID, secondExecution.LineDispatchID,
		"both steps of the same traversal share one line dispatch")

	dispatch, err := models.FindWorkOrderLineDispatch(database.Conn(), firstExecution.LineDispatchID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateActive, dispatch.State,
		"a passed step with a next step in the snapshot leaves the traversal active")
}

// Test__RunFinalizer__ExecuteNextFactoryLineStep__FinishesDispatchOnLastStepPass
// covers acceptance criterion 2: a traversal whose steps all pass finishes
// as passed.
func Test__RunFinalizer__ExecuteNextFactoryLineStep__FinishesDispatchOnLastStepPass(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(database.Conn(), "Ship feature", "", &r.User, nil, nil)
	require.NoError(t, err)
	dispatchWorkOrderForTest(t, order)

	line, err := factory.CreateLine(database.Conn(), "ship", nil)
	require.NoError(t, err)

	onlyApp, onlyEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
	require.NoError(t, line.Update(database.Conn(), nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: onlyApp.ID, Entrypoint: onlyEntry},
	}))

	var dispatch *models.FactoryWorkOrderLineDispatch
	var result *models.FactoryLineStepResult
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		dispatch, result, dispatchErr = line.Dispatch(tx, order)
		return dispatchErr
	}))

	now := time.Now()
	require.NoError(t, database.Conn().Model(result.Run).Updates(map[string]any{
		"state":       models.CanvasRunStateFinished,
		"result":      models.CanvasRunResultPassed,
		"updated_at":  &now,
		"finished_at": &now,
	}).Error)

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)
	var pending []factoryLinePendingRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var advanceErr error
		pending, advanceErr = finalizer.executeNextFactoryLineStep(tx, result.Run.ID)
		return advanceErr
	}))
	assert.Empty(t, pending, "no next step, so nothing to run")

	reloaded, err := models.FindWorkOrderLineDispatch(database.Conn(), dispatch.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, reloaded.State)
	assert.Equal(t, models.CanvasRunResultPassed, reloaded.Result)
	assert.NotNil(t, reloaded.FinishedAt)
}

// Test__RunFinalizer__ExecuteNextFactoryLineStep__FinishesDispatchOnFailure
// and Test__RunFinalizer__ExecuteNextFactoryLineStep__FinishesDispatchOnCancellation
// cover acceptance criterion 2's other two outcomes.
func Test__RunFinalizer__ExecuteNextFactoryLineStep__FinishesDispatchOnFailure(t *testing.T) {
	testExecuteNextFactoryLineStepFinishesDispatchWithResult(t, models.CanvasRunResultFailed)
}

func Test__RunFinalizer__ExecuteNextFactoryLineStep__FinishesDispatchOnCancellation(t *testing.T) {
	testExecuteNextFactoryLineStepFinishesDispatchWithResult(t, models.CanvasRunResultCancelled)
}

func testExecuteNextFactoryLineStepFinishesDispatchWithResult(t *testing.T, terminalResult string) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(database.Conn(), "Ship feature", "", &r.User, nil, nil)
	require.NoError(t, err)
	dispatchWorkOrderForTest(t, order)

	line, err := factory.CreateLine(database.Conn(), "ship", nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-two", "start-two")
	require.NoError(t, line.Update(database.Conn(), nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	}))

	var dispatch *models.FactoryWorkOrderLineDispatch
	var result *models.FactoryLineStepResult
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		dispatch, result, dispatchErr = line.Dispatch(tx, order)
		return dispatchErr
	}))

	now := time.Now()
	require.NoError(t, database.Conn().Model(result.Run).Updates(map[string]any{
		"state":       models.CanvasRunStateFinished,
		"result":      terminalResult,
		"updated_at":  &now,
		"finished_at": &now,
	}).Error)

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)
	var pending []factoryLinePendingRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var advanceErr error
		pending, advanceErr = finalizer.executeNextFactoryLineStep(tx, result.Run.ID)
		return advanceErr
	}))
	assert.Empty(t, pending, "a non-passed result never starts the next step")

	reloaded, err := models.FindWorkOrderLineDispatch(database.Conn(), dispatch.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, reloaded.State)
	assert.Equal(t, terminalResult, reloaded.Result)
}

// A work order can close while a step run is still executing. When that run
// later passes, the traversal must not advance — and it must not stay active
// either, or the reopened order could never dispatch again.
func Test__RunFinalizer__ExecuteNextFactoryLineStep__CancelsDispatchWhenOrderClosedMidTraversal(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(database.Conn(), "Ship feature", "", &r.User, nil, nil)
	require.NoError(t, err)
	dispatchWorkOrderForTest(t, order)

	line, err := factory.CreateLine(database.Conn(), "ship", nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-two", "start-two")
	require.NoError(t, line.Update(database.Conn(), nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	}))

	var dispatch *models.FactoryWorkOrderLineDispatch
	var result *models.FactoryLineStepResult
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		dispatch, result, dispatchErr = line.Dispatch(tx, order)
		return dispatchErr
	}))

	_, err = order.Close(database.Conn(), models.FactoryWorkOrderResultCompleted, &r.User)
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, database.Conn().Model(result.Run).Updates(map[string]any{
		"state":       models.CanvasRunStateFinished,
		"result":      models.CanvasRunResultPassed,
		"updated_at":  &now,
		"finished_at": &now,
	}).Error)

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)
	var pending []factoryLinePendingRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var advanceErr error
		pending, advanceErr = finalizer.executeNextFactoryLineStep(tx, result.Run.ID)
		return advanceErr
	}))
	assert.Empty(t, pending, "a closed order never starts the next step")

	reloaded, err := models.FindWorkOrderLineDispatch(database.Conn(), dispatch.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, reloaded.State,
		"an abandoned traversal must not stay active")
	assert.Equal(t, models.CanvasRunResultCancelled, reloaded.Result)
	assert.NotNil(t, reloaded.FinishedAt)
}

// Test__RunFinalizer__ExecuteNextFactoryLineStep__LineEditMidTraversalDoesNotChangeNextStep
// covers acceptance criterion 3: editing a line's steps while a work order
// is mid-traversal doesn't change which step runs next for that traversal.
func Test__RunFinalizer__ExecuteNextFactoryLineStep__LineEditMidTraversalDoesNotChangeNextStep(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(database.Conn(), "Ship feature", "", &r.User, nil, nil)
	require.NoError(t, err)
	dispatchWorkOrderForTest(t, order)

	line, err := factory.CreateLine(database.Conn(), "ship", nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-two", "start-two")
	require.NoError(t, line.Update(database.Conn(), nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	}))

	var result *models.FactoryLineStepResult
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		_, result, dispatchErr = line.Dispatch(tx, order)
		return dispatchErr
	}))

	// Edit the line mid-traversal: insert a new first step and rename the
	// steps the snapshot already captured. The live line no longer agrees
	// with the dispatch's snapshot about what "step two" is.
	insertedApp, insertedEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "inserted", "start-inserted")
	require.NoError(t, line.Update(database.Conn(), nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: insertedApp.ID, Entrypoint: insertedEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	}))

	now := time.Now()
	require.NoError(t, database.Conn().Model(result.Run).Updates(map[string]any{
		"state":       models.CanvasRunStateFinished,
		"result":      models.CanvasRunResultPassed,
		"updated_at":  &now,
		"finished_at": &now,
	}).Error)

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)
	var pending []factoryLinePendingRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var advanceErr error
		pending, advanceErr = finalizer.executeNextFactoryLineStep(tx, result.Run.ID)
		return advanceErr
	}))

	require.Len(t, pending, 1)
	assert.Equal(t, secondApp.ID, pending[0].workflowID,
		"advancement reads the dispatch's snapshot, not the edited live line")

	secondExecution, err := models.FindWorkOrderExecutionByRunID(database.Conn(), pending[0].runID)
	require.NoError(t, err)
	assert.Equal(t, 1, secondExecution.StepIndex)
	assert.Equal(t, secondApp.Name, secondExecution.StepName,
		"the snapshot's original automation name, unaffected by the later line edit")
}

// dispatchWorkOrderForTest promotes a draft order to open, mirroring
// the dispatch API — needed by tests that poke `StartStep` directly.
func dispatchWorkOrderForTest(t *testing.T, order *models.FactoryWorkOrder) {
	t.Helper()
	_, err := order.UpdateStatus(database.Conn(), models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)
}

func Test__RunFinalizer__FinalizeRunAdvancesFactoryLineInSameTransaction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(database.Conn(), "Ship feature", "", &r.User, nil, nil)
	require.NoError(t, err)
	dispatchWorkOrderForTest(t, order)

	line, err := factory.CreateLine(database.Conn(), "ship", nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-two", "start-two")

	steps := []models.FactoryLineStep{
		{
			Type:       models.FactoryLineStepTypeRunApp,
			AppID:      firstApp.ID,
			Entrypoint: firstEntry,
		},
		{
			Type:       models.FactoryLineStepTypeRunApp,
			AppID:      secondApp.ID,
			Entrypoint: secondEntry,
		},
	}
	require.NoError(t, line.Update(database.Conn(), nil, steps))

	var firstResult *models.FactoryLineStepResult
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var startErr error
		_, firstResult, startErr = line.Dispatch(tx, order)
		return startErr
	}))

	require.NoError(t, database.Conn().Model(firstResult.Run).Updates(map[string]any{
		"state": models.CanvasRunStateStarted,
	}).Error)

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)
	require.NoError(t, finalizer.finalizeRun(firstApp.ID, firstResult.Run.ID, runFinalizerTriggerEventTerminal))

	updatedRun, err := models.FindCanvasRunInTransaction(database.Conn(), firstApp.ID, firstResult.Run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultPassed, updatedRun.Result)

	firstExecution, err := models.FindWorkOrderExecutionByRunID(database.Conn(), firstResult.Run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusFinished, firstExecution.Status)

	var secondRun models.CanvasRun
	require.NoError(t, database.Conn().
		Where("workflow_id = ? AND state = ?", secondApp.ID, models.CanvasRunStatePending).
		First(&secondRun).Error)

	secondExecution, err := models.FindWorkOrderExecutionByRunID(database.Conn(), secondRun.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, secondExecution.StepIndex)
	assert.Equal(t, secondApp.Name, secondExecution.StepName)
}

func Test__RunFinalizer__FinalizeRunRollsBackWhenFactoryLineAdvanceFails(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(database.Conn(), "Ship feature", "", &r.User, nil, nil)
	require.NoError(t, err)
	dispatchWorkOrderForTest(t, order)

	line, err := factory.CreateLine(database.Conn(), "ship", nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
	secondApp, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-two", "start-two")

	steps := []models.FactoryLineStep{
		{
			Type:       models.FactoryLineStepTypeRunApp,
			AppID:      firstApp.ID,
			Entrypoint: firstEntry,
		},
		{
			Type:       models.FactoryLineStepTypeRunApp,
			AppID:      secondApp.ID,
			Entrypoint: "missing-entrypoint",
		},
	}
	require.NoError(t, line.Update(database.Conn(), nil, steps))

	var firstResult *models.FactoryLineStepResult
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var startErr error
		_, firstResult, startErr = line.Dispatch(tx, order)
		return startErr
	}))

	require.NoError(t, database.Conn().Model(firstResult.Run).Updates(map[string]any{
		"state": models.CanvasRunStateStarted,
	}).Error)

	amqpURL, _ := config.RabbitMQURL()
	finalizer := NewRunFinalizer(amqpURL, r.Registry)
	err = finalizer.finalizeRun(firstApp.ID, firstResult.Run.ID, runFinalizerTriggerEventTerminal)
	require.Error(t, err)

	updatedRun, err := models.FindCanvasRunInTransaction(database.Conn(), firstApp.ID, firstResult.Run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, updatedRun.State)

	firstExecution, err := models.FindWorkOrderExecutionByRunID(database.Conn(), firstResult.Run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, firstExecution.Status)
}
