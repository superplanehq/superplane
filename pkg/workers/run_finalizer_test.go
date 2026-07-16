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
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
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
