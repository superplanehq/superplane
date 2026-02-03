package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ResolveExecutionErrors__ResolvesMultipleExecutions(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

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
	executionOne := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID, nil)
	executionTwo := support.CreateCanvasNodeExecution(t, canvas.ID, "node-2", rootEvent.ID, rootEvent.ID, nil)

	require.NoError(t, executionOne.Fail(models.CanvasNodeExecutionResultReasonError, "boom"))
	require.NoError(t, executionTwo.Fail(models.CanvasNodeExecutionResultReasonError, "boom"))

	response, err := ResolveExecutionErrors(context.Background(), canvas.ID, []uuid.UUID{
		executionOne.ID,
		executionTwo.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, response)

	updatedExecutionOne, err := models.FindNodeExecution(canvas.ID, executionOne.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionResultReasonErrorResolved, updatedExecutionOne.ResultReason)

	updatedExecutionTwo, err := models.FindNodeExecution(canvas.ID, executionTwo.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionResultReasonErrorResolved, updatedExecutionTwo.ResultReason)
}

func Test__ResolveExecutionErrors__ReturnsErrorForNonErrorExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

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
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID, nil)

	_, err := ResolveExecutionErrors(context.Background(), canvas.ID, []uuid.UUID{execution.ID})
	require.Error(t, err)

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, "", updatedExecution.ResultReason)
}
