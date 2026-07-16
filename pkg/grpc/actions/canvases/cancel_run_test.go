package canvases

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
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

func Test__CancelRun__CancelsActiveWorkAndFinalizesRun(t *testing.T) {
	r := support.Setup(t)

	amqpURL, _ := config.RabbitMQURL()
	runConsumer := testconsumer.New(amqpURL, messages.CanvasRunRoutingKey)
	runConsumer.Start()
	defer runConsumer.Stop()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{NodeID: "node-2", Name: "Node 2", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := cancelRunTestCreateRun(t, rootEvent)
	execution := cancelRunTestCreateExecution(t, run, rootEvent.ID, "node-1", models.CanvasNodeExecutionStateStarted)

	queueItem := support.CreateQueueItem(t, canvas.ID, "node-2", rootEvent.ID, rootEvent.ID)
	queueItem.RunID = run.ID
	require.NoError(t, database.Conn().Save(queueItem).Error)

	response, err := CancelRun(context.Background(), r.AuthService, r.Encryptor, r.Organization.ID.String(), r.Registry, canvas.ID, run.ID)
	require.NoError(t, err)
	require.NotNil(t, response)

	updatedRun, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultCancelled, updatedRun.Result)
	require.NotNil(t, updatedRun.CancelledAt)

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)

	openWork, err := models.FindOpenCanvasRunWorkInTransaction(database.Conn(), run.ID)
	require.NoError(t, err)
	assert.False(t, openWork.HasActiveExecutions)
	assert.False(t, openWork.HasQueueItems)

	assert.True(t, runConsumer.HasReceivedMessage())
}

func Test__CancelRun__IdempotentOnFinishedRun(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{NodeID: "node-1", Name: "Node 1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := cancelRunTestCreateRun(t, rootEvent)

	now := time.Now()
	require.NoError(t, database.Conn().Model(run).Updates(map[string]any{
		"state":       models.CanvasRunStateFinished,
		"result":      models.CanvasRunResultPassed,
		"finished_at": &now,
		"updated_at":  &now,
	}).Error)

	response, err := CancelRun(context.Background(), r.AuthService, r.Encryptor, r.Organization.ID.String(), r.Registry, canvas.ID, run.ID)
	require.NoError(t, err)
	require.NotNil(t, response)

	updatedRun, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultPassed, updatedRun.Result)
	assert.Nil(t, updatedRun.CancelledAt)
}

func Test__CancelRun__ReturnsNotFoundForNonExistentRun(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)

	_, err := CancelRun(context.Background(), r.AuthService, r.Encryptor, r.Organization.ID.String(), r.Registry, canvas.ID, uuid.New())
	require.Error(t, err)
}

func cancelRunTestCreateRun(t *testing.T, rootEvent *models.CanvasEvent) *models.CanvasRun {
	var run *models.CanvasRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		run, err = models.FindOrCreateCanvasRunForRootEventInTransaction(tx, rootEvent)
		if err != nil {
			return err
		}

		return rootEvent.RoutedInTransaction(tx)
	}))

	return run
}

func cancelRunTestCreateExecution(t *testing.T, run *models.CanvasRun, rootEventID uuid.UUID, nodeID, state string) *models.CanvasNodeExecution {
	now := time.Now()
	execution := models.CanvasNodeExecution{
		ID:            uuid.New(),
		WorkflowID:    run.WorkflowID,
		NodeID:        nodeID,
		RootEventID:   rootEventID,
		RunID:         run.ID,
		EventID:       rootEventID,
		State:         state,
		Configuration: datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}
