package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__EventRouter_ProcessRootEvent(t *testing.T) {
	amqpURL, _ := config.RabbitMQURL()

	router := NewEventRouter(amqpURL)
	logger := log.NewEntry(log.New())
	r := support.Setup(t)

	queueConsumer := testconsumer.New(amqpURL, messages.CanvasQueueItemCreatedRoutingKey)
	queueConsumer.Start()
	defer queueConsumer.Stop()

	runConsumer := testconsumer.New(amqpURL, messages.CanvasRunRoutingKey)
	runConsumer.Start()
	defer runConsumer.Stop()

	terminalEventConsumer := testconsumer.New(amqpURL, messages.EventTerminalRoutingKey)
	terminalEventConsumer.Start()
	defer terminalEventConsumer.Stop()

	//
	// Create a simple canvas with just a trigger and a component nodes.
	//
	node1 := "trigger-1"
	node2 := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: node1, Type: models.NodeTypeTrigger},
			{NodeID: node2, Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: node1, TargetID: node2, Channel: "default"},
		},
	)

	//
	// Create the root event for the trigger node, and process it.
	//
	event := support.EmitCanvasEventForNode(t, canvas.ID, node1, "default", nil)
	err := router.LockAndProcessEvent(logger, *event, time.Now())
	require.NoError(t, err)

	//
	// Verify event was marked as routed and
	// new queue item was created for the target node.
	//
	updatedEvent, err := models.FindCanvasEvent(event.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasEventStateRouted, updatedEvent.State)
	assert.NotEqual(t, uuid.Nil, updatedEvent.RunID)

	run, err := models.FindCanvasRunByRootEventInTransaction(database.Conn(), event.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, run.State)

	queueItems, err := models.ListNodeQueueItems(canvas.ID, node2, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)
	assert.Equal(t, node2, queueItems[0].NodeID)
	assert.Equal(t, event.ID, queueItems[0].EventID)
	assert.Equal(t, run.ID, queueItems[0].RunID)

	assert.True(t, queueConsumer.HasReceivedMessage())
	assert.True(t, runConsumer.HasReceivedMessage())
	assert.False(t, terminalEventConsumer.HasReceivedMessage())
}

func Test__EventRouter_DoesNotRouteEventForSoftDeletedOrganization(t *testing.T) {
	amqpURL, _ := config.RabbitMQURL()

	router := NewEventRouter(amqpURL)
	logger := log.NewEntry(log.New())
	r := support.Setup(t)

	queueConsumer := testconsumer.New(amqpURL, messages.CanvasQueueItemCreatedRoutingKey)
	queueConsumer.Start()
	defer queueConsumer.Stop()

	triggerNode := "trigger-1"
	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNode, Type: models.NodeTypeTrigger},
			{NodeID: componentNode, Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))

	events, err := models.ListPendingCanvasEvents()
	require.NoError(t, err)
	for _, pending := range events {
		assert.NotEqual(t, event.ID, pending.ID)
	}

	require.NoError(t, router.LockAndProcessEvent(logger, *event, time.Now()))

	updatedEvent, err := models.FindCanvasEvent(event.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasEventStatePending, updatedEvent.State)

	queueItems, err := models.ListNodeQueueItems(canvas.ID, componentNode, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, queueItems)

	assert.False(t, queueConsumer.HasReceivedMessage())
}

func Test__EventRouter_ProcessExecutionEvent(t *testing.T) {
	amqpURL, _ := config.RabbitMQURL()

	router := NewEventRouter(amqpURL)
	logger := log.NewEntry(log.New())
	r := support.Setup(t)

	queueConsumer := testconsumer.New(amqpURL, messages.CanvasQueueItemCreatedRoutingKey)
	queueConsumer.Start()
	defer queueConsumer.Stop()

	runConsumer := testconsumer.New(amqpURL, messages.CanvasRunRoutingKey)
	runConsumer.Start()
	defer runConsumer.Stop()

	trigger1 := "trigger-1"
	node1 := "component-1"
	node2 := "component-2"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: trigger1, Type: models.NodeTypeTrigger},
			{NodeID: node1, Type: models.NodeTypeComponent},
			{NodeID: node2, Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: trigger1, TargetID: node1, Channel: "default"},
			{SourceID: node1, TargetID: node2, Channel: "default"},
		},
	)

	//
	// Create root event for trigger node,
	// and create execution with output event for node1.
	//
	triggerEvent := support.EmitCanvasEventForNode(t, canvas.ID, trigger1, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), triggerEvent)
	require.NoError(t, err)
	require.NoError(t, triggerEvent.Routed())
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, node1, triggerEvent.ID, triggerEvent.ID)
	execution.RunID = run.ID
	require.NoError(t, database.Conn().Save(execution).Error)
	_, err = execution.Pass(map[string][]any{"default": {map[string]any{}}})
	require.NoError(t, err)

	//
	// Process node1 output event and verify it was routed properly.
	//
	events, err := models.ListCanvasEvents(canvas.ID, node1, 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	outputEvent := events[0]
	err = router.LockAndProcessEvent(logger, outputEvent, time.Now())
	require.NoError(t, err)

	updatedEvent, err := models.FindCanvasEvent(outputEvent.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasEventStateRouted, updatedEvent.State)

	queueItems, err := models.ListNodeQueueItems(canvas.ID, node2, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)
	assert.Equal(t, node2, queueItems[0].NodeID)
	assert.Equal(t, outputEvent.ID, queueItems[0].EventID)

	assert.True(t, queueConsumer.HasReceivedMessage())
	assert.False(t, runConsumer.HasReceivedMessage())
}

func Test__EventRouter_ProcessExecutionEventUsesRunVersionEdges(t *testing.T) {
	amqpURL, _ := config.RabbitMQURL()

	router := NewEventRouter(amqpURL)
	logger := log.NewEntry(log.New())
	r := support.Setup(t)

	trigger := "trigger-1"
	node1 := "component-1"
	node2 := "component-2"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: trigger, Type: models.NodeTypeTrigger},
			{NodeID: node1, Type: models.NodeTypeComponent},
			{NodeID: node2, Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: trigger, TargetID: node1, Channel: "default"},
			{SourceID: node1, TargetID: node2, Channel: "default"},
		},
	)

	triggerEvent := support.EmitCanvasEventForNode(t, canvas.ID, trigger, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), triggerEvent)
	require.NoError(t, err)
	require.NoError(t, triggerEvent.Routed())

	publishCanvasVersionWithEdges(t, canvas.ID, []models.Edge{
		{SourceID: trigger, TargetID: node1, Channel: "default"},
	})

	execution := support.CreateCanvasNodeExecution(t, canvas.ID, node1, triggerEvent.ID, triggerEvent.ID)
	execution.RunID = run.ID
	require.NoError(t, database.Conn().Save(execution).Error)
	_, err = execution.Pass(map[string][]any{"default": {map[string]any{}}})
	require.NoError(t, err)

	events, err := models.ListCanvasEvents(canvas.ID, node1, 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)

	err = router.LockAndProcessEvent(logger, events[0], time.Now())
	require.NoError(t, err)

	queueItems, err := models.ListNodeQueueItems(canvas.ID, node2, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)
	assert.Equal(t, run.ID, queueItems[0].RunID)
}

func Test__EventRouter_ProcessTerminalExecutionEventFinishesRun(t *testing.T) {
	amqpURL, _ := config.RabbitMQURL()

	router := NewEventRouter(amqpURL)
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

	finalizer := NewRunFinalizer(amqpURL)
	require.NoError(t, finalizer.finalizeRun(canvas.ID, run.ID, runFinalizerTriggerEventTerminal))

	updatedRun, err = models.FindCanvasRunByRootEventInTransaction(database.Conn(), triggerEvent.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultPassed, updatedRun.Result)
	assert.NotNil(t, updatedRun.FinishedAt)
}

func publishCanvasVersionWithEdges(t *testing.T, canvasID uuid.UUID, edges []models.Edge) {
	t.Helper()

	tx := database.Conn()
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(tx, canvasID)
	require.NoError(t, err)

	now := time.Now()
	versionID := uuid.New()
	version := models.CanvasVersion{
		ID:          versionID,
		WorkflowID:  canvasID,
		State:       models.CanvasVersionStatePublished,
		PublishedAt: &now,
		Nodes:       liveVersion.Nodes,
		Edges:       datatypes.NewJSONSlice(edges),
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}

	require.NoError(t, tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&version).Error; err != nil {
			return err
		}

		return tx.Model(&models.Canvas{}).
			Where("id = ?", canvasID).
			Update("live_version_id", versionID).
			Error
	}))
}
