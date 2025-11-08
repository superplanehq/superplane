package workflows

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ListEventExecutions__ReturnsEmptyListWhenNoExecutionsExist(t *testing.T) {
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

	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)

	response, err := ListEventExecutions(context.Background(), r.Registry, workflow.ID.String(), rootEvent.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Empty(t, response.Executions)
}

func Test__ListEventExecutions__ReturnsParentExecutionsForEvent(t *testing.T) {
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

	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
	event := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)

	parentExecution := support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", rootEvent.ID, event.ID, nil)

	response, err := ListEventExecutions(context.Background(), r.Registry, workflow.ID.String(), rootEvent.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Executions, 1)

	execution := response.Executions[0]
	assert.Equal(t, parentExecution.ID.String(), execution.Id)
	assert.Equal(t, workflow.ID.String(), execution.WorkflowId)
	assert.Equal(t, "node-1", execution.NodeId)
	assert.Empty(t, execution.ParentExecutionId)
	assert.Empty(t, execution.ChildExecutions)
}

func Test__ListEventExecutions__ReturnsParentExecutionsWithChildExecutions(t *testing.T) {
	r := support.Setup(t)

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeBlueprint,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Blueprint: &models.BlueprintRef{ID: "test-blueprint"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
	event := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)

	parentExecution := support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", rootEvent.ID, event.ID, nil)

	childEvent1 := support.EmitWorkflowEventForNode(t, workflow.ID, "child-node-1", "default", nil)
	childEvent2 := support.EmitWorkflowEventForNode(t, workflow.ID, "child-node-2", "default", nil)

	childExecution1 := support.CreateWorkflowNodeExecution(t, workflow.ID, "child-node-1", rootEvent.ID, childEvent1.ID, &parentExecution.ID)
	childExecution2 := support.CreateWorkflowNodeExecution(t, workflow.ID, "child-node-2", rootEvent.ID, childEvent2.ID, &parentExecution.ID)

	response, err := ListEventExecutions(context.Background(), r.Registry, workflow.ID.String(), rootEvent.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Executions, 1)

	execution := response.Executions[0]
	assert.Equal(t, parentExecution.ID.String(), execution.Id)
	assert.Equal(t, workflow.ID.String(), execution.WorkflowId)
	assert.Equal(t, "node-1", execution.NodeId)
	assert.Empty(t, execution.ParentExecutionId)

	require.Len(t, execution.ChildExecutions, 2)

	childExecutionIDs := []string{execution.ChildExecutions[0].Id, execution.ChildExecutions[1].Id}
	assert.Contains(t, childExecutionIDs, childExecution1.ID.String())
	assert.Contains(t, childExecutionIDs, childExecution2.ID.String())

	for _, childExec := range execution.ChildExecutions {
		assert.Equal(t, workflow.ID.String(), childExec.WorkflowId)
		assert.Equal(t, parentExecution.ID.String(), childExec.ParentExecutionId)
		assert.Empty(t, childExec.ChildExecutions)
	}
}

func Test__ListEventExecutions__OnlyReturnsExecutionsForSpecificRootEvent(t *testing.T) {
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

	event1 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
	event2 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)

	execution1 := support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", rootEvent1.ID, event1.ID, nil)
	support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", rootEvent2.ID, event2.ID, nil)

	response, err := ListEventExecutions(context.Background(), r.Registry, workflow.ID.String(), rootEvent1.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Executions, 1)

	assert.Equal(t, execution1.ID.String(), response.Executions[0].Id)
}
