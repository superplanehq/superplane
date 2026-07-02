package workers

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
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
