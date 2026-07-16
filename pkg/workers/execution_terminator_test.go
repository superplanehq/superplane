package workers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ExecutionTerminator__CancelsActiveExecution(t *testing.T) {
	r := support.Setup(t)

	amqpURL, _ := config.RabbitMQURL()
	executionFinishedConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionFinishedRoutingKey)
	executionFinishedConsumer.Start()
	defer executionFinishedConsumer.Stop()

	terminator := NewExecutionTerminator(amqpURL, r.AuthService, r.Encryptor, r.Registry)

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

	require.NoError(t, execution.RequestCancellation(database.DB(t.Context()), &r.User))
	require.NoError(t, terminator.LockAndCancelExecution(execution.ID))

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)

	require.NoError(t, terminator.LockAndCancelExecution(execution.ID))
	assert.True(t, executionFinishedConsumer.HasReceivedMessage())
}
