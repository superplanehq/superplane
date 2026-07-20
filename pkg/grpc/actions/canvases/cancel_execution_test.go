package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__CancelExecution__RequestsCancellation(t *testing.T) {
	r := support.Setup(t)

	amqpURL, _ := config.RabbitMQURL()
	cancellingConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionCancellingRoutingKey)
	cancellingConsumer.Start()
	defer cancellingConsumer.Stop()

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
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	response, err := CancelExecution(ctx, r.AuthService, r.Encryptor, r.Organization.ID.String(), r.Registry, canvas.ID, execution.ID)
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.True(t, cancellingConsumer.HasReceivedMessage())

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)
	assert.NotNil(t, updatedExecution.CancelledAt)
}

func Test__CancelExecution__IsIdempotent(t *testing.T) {
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
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	_, err := CancelExecution(ctx, r.AuthService, r.Encryptor, r.Organization.ID.String(), r.Registry, canvas.ID, execution.ID)
	require.NoError(t, err)

	_, err = CancelExecution(ctx, r.AuthService, r.Encryptor, r.Organization.ID.String(), r.Registry, canvas.ID, execution.ID)
	require.NoError(t, err)

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)
}

func Test__CancelExecution__ReturnsNotFoundForNonExistentExecution(t *testing.T) {
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

	nonExistentID := uuid.New()
	_, err := CancelExecution(context.Background(), r.AuthService, r.Encryptor, r.Organization.ID.String(), r.Registry, canvas.ID, nonExistentID)
	require.Error(t, err)
}
