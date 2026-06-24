package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

func Test__ListEventExecutions__ReturnsErrorForInvalidCanvasID(t *testing.T) {
	r := support.Setup(t)

	response, err := ListEventExecutions(context.Background(), r.Registry, "invalid-uuid", uuid.New().String())
	require.Error(t, err)
	assert.Nil(t, response)
	code, _, ok := grpcerrors.HandlerStatus(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, code)
}

func Test__ListEventExecutions__ReturnsErrorForInvalidEventID(t *testing.T) {
	r := support.Setup(t)

	response, err := ListEventExecutions(context.Background(), r.Registry, uuid.New().String(), "bogus")
	require.Error(t, err)
	assert.Nil(t, response)
	code, _, ok := grpcerrors.HandlerStatus(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, code)
}

func Test__ListEventExecutions__ReturnsEmptyListWhenNoExecutionsExist(t *testing.T) {
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

	response, err := ListEventExecutions(context.Background(), r.Registry, canvas.ID.String(), rootEvent.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Empty(t, response.Executions)
}

func Test__ListEventExecutions__ReturnsParentExecutionsForEvent(t *testing.T) {
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
	event := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)

	parentExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, event.ID)

	response, err := ListEventExecutions(context.Background(), r.Registry, canvas.ID.String(), rootEvent.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Executions, 1)

	execution := response.Executions[0]
	assert.Equal(t, parentExecution.ID.String(), execution.Id)
	assert.Equal(t, canvas.ID.String(), execution.CanvasId)
	assert.Equal(t, "node-1", execution.NodeId)
}

func Test__ListEventExecutions__OnlyReturnsExecutionsForSpecificRootEvent(t *testing.T) {
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

	rootEvent1 := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	rootEvent2 := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)

	event1 := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	event2 := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)

	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent1.ID, event1.ID)
	support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent2.ID, event2.ID)

	response, err := ListEventExecutions(context.Background(), r.Registry, canvas.ID.String(), rootEvent1.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Executions, 1)

	assert.Equal(t, execution1.ID.String(), response.Executions[0].Id)
}
