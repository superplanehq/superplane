package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

func Test_DeleteNodeQueueItem_ReturnsErrorForInvalidCanvasID(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	response, err := DeleteNodeQueueItem(context.Background(), r.Registry, "invalid-uuid", "node-1", uuid.New().String())
	require.Error(t, err)
	assert.Nil(t, response)
	code, _, ok := grpcerrors.HandlerStatus(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, code)
}

func Test_DeleteNodeQueueItem_ReturnsErrorForInvalidItemID(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	response, err := DeleteNodeQueueItem(context.Background(), r.Registry, uuid.New().String(), "node-1", "bogus")
	require.Error(t, err)
	assert.Nil(t, response)
	code, _, ok := grpcerrors.HandlerStatus(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, code)
}

func Test_DeleteNodeQueueItem(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	nodeID := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: nodeID,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		nil,
	)

	// Create a single queue item on that node
	event := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	support.CreateQueueItem(t, canvas.ID, nodeID, event.ID, event.ID)

	items, err := models.ListNodeQueueItems(canvas.ID, nodeID, 10, nil)
	require.NoError(t, err)
	require.Len(t, items, 1)

	_, err = DeleteNodeQueueItem(context.Background(), r.Registry, canvas.ID.String(), nodeID, items[0].ID.String())
	require.NoError(t, err)

	remaining, err := models.ListNodeQueueItems(canvas.ID, nodeID, 10, nil)
	require.NoError(t, err)
	require.Len(t, remaining, 0)
}

func Test_DeleteNodeQueueItem_RequestsRunFinalization(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	finalizationConsumer := testconsumer.New(amqpURL, messages.CanvasQueueItemDeletedRoutingKey)
	finalizationConsumer.Start()
	defer finalizationConsumer.Stop()

	nodeID := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: nodeID,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		nil,
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	require.NoError(t, event.Routed())
	queueItem := support.CreateQueueItem(t, canvas.ID, nodeID, event.ID, event.ID)

	_, err := DeleteNodeQueueItem(context.Background(), r.Registry, canvas.ID.String(), nodeID, queueItem.ID.String())
	require.NoError(t, err)
	assert.True(t, finalizationConsumer.HasReceivedMessage())

	run, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, event.RunID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, run.State)
}
