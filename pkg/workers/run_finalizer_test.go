package workers

import (
	"testing"
	"time"

	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
)

func Test__RunFinalizer_FinalizesRunAfterTerminalExecutionEvent(t *testing.T) {
	amqpURL, _ := config.RabbitMQURL()

	router := NewEventRouter(amqpURL)
	finalizer := NewRunFinalizer(amqpURL)
	logger := log.NewEntry(log.New())
	r := support.Setup(t)

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

	events, err := models.ListCanvasEvents(canvas.ID, node, 10, nil)
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
	amqpURL, _ := config.RabbitMQURL()

	finalizer := NewRunFinalizer(amqpURL)
	r := support.Setup(t)

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
	amqpURL, _ := config.RabbitMQURL()

	finalizer := NewRunFinalizer(amqpURL)
	r := support.Setup(t)

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
	amqpURL, _ := config.RabbitMQURL()

	finalizer := NewRunFinalizer(amqpURL)
	r := support.Setup(t)

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
	amqpURL, _ := config.RabbitMQURL()

	finalizer := NewRunFinalizer(amqpURL)
	r := support.Setup(t)

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
	amqpURL, _ := config.RabbitMQURL()

	finalizer := NewRunFinalizer(amqpURL)
	r := support.Setup(t)

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
	amqpURL, _ := config.RabbitMQURL()

	finalizer := NewRunFinalizer(amqpURL)
	r := support.Setup(t)

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
	amqpURL, _ := config.RabbitMQURL()

	finalizer := NewRunFinalizer(amqpURL)
	r := support.Setup(t)

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
