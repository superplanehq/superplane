package canvases

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__CancelRun__RequestsCancellationAndDrainsWork(t *testing.T) {
	r := support.Setup(t)

	amqpURL, _ := config.RabbitMQURL()
	runConsumer := testconsumer.New(amqpURL, messages.CanvasRunRoutingKey)
	runConsumer.Start()
	defer runConsumer.Stop()

	executionCancellingConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionCancellingRoutingKey)
	executionCancellingConsumer.Start()
	defer executionCancellingConsumer.Stop()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{
				NodeID: "node-2",
				Name:   "Node 2",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), rootEvent)
	require.NoError(t, err)
	require.NoError(t, rootEvent.Routed())

	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID)
	execution.RunID = run.ID
	execution.State = models.CanvasNodeExecutionStateStarted
	require.NoError(t, database.Conn().Save(execution).Error)

	queueItem := models.CanvasNodeQueueItem{
		WorkflowID:  canvas.ID,
		NodeID:      "node-2",
		RootEventID: rootEvent.ID,
		RunID:       run.ID,
		EventID:     rootEvent.ID,
	}
	require.NoError(t, database.Conn().Create(&queueItem).Error)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	response, err := CancelRun(ctx, r.Organization.ID.String(), canvas.ID, run.ID)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Run)
	assert.Equal(t, pb.CanvasRun_STATE_CANCELLING, response.Run.State)
	assert.True(t, runConsumer.HasReceivedMessage())
	assert.True(t, executionCancellingConsumer.HasReceivedMessage())

	updatedRun, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, updatedRun.State)
	assert.NotNil(t, updatedRun.CancelledAt)

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)

	var queueItemCount int64
	require.NoError(t, database.Conn().Model(&models.CanvasNodeQueueItem{}).Where("run_id = ?", run.ID).Count(&queueItemCount).Error)
	assert.Zero(t, queueItemCount)
}

func Test__CancelRun__IsIdempotentForCancellingRun(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), rootEvent)
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, database.Conn().Model(run).Updates(map[string]any{
		"state":        models.CanvasRunStateCancelling,
		"cancelled_at": now,
		"cancelled_by": r.User,
	}).Error)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	_, err = CancelRun(ctx, r.Organization.ID.String(), canvas.ID, run.ID)
	require.NoError(t, err)

	_, err = CancelRun(ctx, r.Organization.ID.String(), canvas.ID, run.ID)
	require.NoError(t, err)

	updatedRun, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, updatedRun.State)
}

func Test__CancelRun__NoOpForFinishedRun(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), rootEvent)
	require.NoError(t, err)

	finishedAt := time.Now()
	require.NoError(t, database.Conn().Model(run).Updates(map[string]any{
		"state":       models.CanvasRunStateFinished,
		"result":      models.CanvasRunResultPassed,
		"finished_at": finishedAt,
	}).Error)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	response, err := CancelRun(ctx, r.Organization.ID.String(), canvas.ID, run.ID)
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, pb.CanvasRun_STATE_FINISHED, response.Run.State)
}

func Test__CancelRun__ReturnsNotFoundForMissingRun(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{},
		[]models.Edge{},
	)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	_, err := CancelRun(ctx, r.Organization.ID.String(), canvas.ID, uuid.New())
	require.Error(t, err)
}
