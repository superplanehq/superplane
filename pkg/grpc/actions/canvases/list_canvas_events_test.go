package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ListCanvasEvents__ReturnsEventsWithExecutions(t *testing.T) {
	r := support.Setup(t)

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
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

	rootEvent1 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
	rootEvent2 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
	customName := "Custom Run Name"
	rootEvent1.CustomName = &customName
	require.NoError(t, database.Conn().Save(rootEvent1).Error)

	parentExecution := support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", rootEvent1.ID, rootEvent1.ID, nil)
	nextExecution := support.CreateNextNodeExecution(t, workflow.ID, "node-1", rootEvent1.ID, rootEvent1.ID, &parentExecution.ID)

	response, err := ListCanvasEvents(context.Background(), r.Registry, workflow.ID, 0, nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Events, 2)

	event1 := findCanvasEventWithExecutions(response.Events, rootEvent1.ID.String())
	require.NotNil(t, event1)
	require.Len(t, event1.Executions, 2)
	assert.Equal(t, customName, event1.CustomName)

	parent := findCanvasNodeExecution(event1.Executions, parentExecution.ID.String())
	require.NotNil(t, parent)
	assert.Equal(t, workflow.ID.String(), parent.CanvasId)
	assert.Equal(t, "node-1", parent.NodeId)
	assert.Empty(t, parent.ParentExecutionId)
	assert.Empty(t, parent.PreviousExecutionId)
	assert.Equal(t, pb.CanvasNodeExecution_STATE_PENDING, parent.State)

	next := findCanvasNodeExecution(event1.Executions, nextExecution.ID.String())
	require.NotNil(t, next)
	assert.Equal(t, workflow.ID.String(), next.CanvasId)
	assert.Equal(t, "node-1", next.NodeId)
	assert.Empty(t, next.ParentExecutionId)
	assert.Equal(t, parentExecution.ID.String(), next.PreviousExecutionId)
	assert.Equal(t, pb.CanvasNodeExecution_STATE_PENDING, next.State)

	event2 := findCanvasEventWithExecutions(response.Events, rootEvent2.ID.String())
	require.NotNil(t, event2)
	assert.Empty(t, event2.Executions)
}

func findCanvasEventWithExecutions(events []*pb.CanvasEventWithExecutions, id string) *pb.CanvasEventWithExecutions {
	for _, event := range events {
		if event.Id == id {
			return event
		}
	}

	return nil
}

func findCanvasNodeExecution(executions []*pb.CanvasNodeExecution, id string) *pb.CanvasNodeExecution {
	for _, execution := range executions {
		if execution.Id == id {
			return execution
		}
	}

	return nil
}
