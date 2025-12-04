package workflows

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

func Test__CancelExecution__CancelsExecutionSuccessfully(t *testing.T) {
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
	execution := support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", rootEvent.ID, rootEvent.ID, nil)

	response, err := CancelExecution(context.Background(), r.Registry, workflow.ID, execution.ID)
	require.NoError(t, err)
	require.NotNil(t, response)

	updatedExecution, err := models.FindNodeExecution(workflow.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.WorkflowNodeExecutionResultCancelled, updatedExecution.Result)
}

func Test__CancelExecution__ReturnsNotFoundForNonExistentExecution(t *testing.T) {
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

	nonExistentID := uuid.New()
	_, err := CancelExecution(context.Background(), r.Registry, workflow.ID, nonExistentID)
	require.Error(t, err)
}

func Test__CancelExecution__CancelsParentExecutionWhenCancellingChild(t *testing.T) {
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
	parentExecution := support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", rootEvent.ID, rootEvent.ID, nil)

	childEvent := support.EmitWorkflowEventForNode(t, workflow.ID, "child-node-1", "default", nil)
	childExecution := support.CreateWorkflowNodeExecution(t, workflow.ID, "child-node-1", rootEvent.ID, childEvent.ID, &parentExecution.ID)

	response, err := CancelExecution(context.Background(), r.Registry, workflow.ID, childExecution.ID)
	require.NoError(t, err)
	require.NotNil(t, response)

	updatedChildExecution, err := models.FindNodeExecution(workflow.ID, childExecution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeExecutionStateFinished, updatedChildExecution.State)
	assert.Equal(t, models.WorkflowNodeExecutionResultCancelled, updatedChildExecution.Result)

	updatedParentExecution, err := models.FindNodeExecution(workflow.ID, parentExecution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeExecutionStateFinished, updatedParentExecution.State)
	assert.Equal(t, models.WorkflowNodeExecutionResultCancelled, updatedParentExecution.Result)
}

func Test__CancelExecution__CancelsAllChildrenWhenCancellingParent(t *testing.T) {
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
	parentExecution := support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", rootEvent.ID, rootEvent.ID, nil)

	childEvent1 := support.EmitWorkflowEventForNode(t, workflow.ID, "child-node-1", "default", nil)
	childEvent2 := support.EmitWorkflowEventForNode(t, workflow.ID, "child-node-2", "default", nil)

	childExecution1 := support.CreateWorkflowNodeExecution(t, workflow.ID, "child-node-1", rootEvent.ID, childEvent1.ID, &parentExecution.ID)
	childExecution2 := support.CreateWorkflowNodeExecution(t, workflow.ID, "child-node-2", rootEvent.ID, childEvent2.ID, &parentExecution.ID)

	response, err := CancelExecution(context.Background(), r.Registry, workflow.ID, parentExecution.ID)
	require.NoError(t, err)
	require.NotNil(t, response)

	updatedParentExecution, err := models.FindNodeExecution(workflow.ID, parentExecution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeExecutionStateFinished, updatedParentExecution.State)
	assert.Equal(t, models.WorkflowNodeExecutionResultCancelled, updatedParentExecution.Result)

	updatedChildExecution1, err := models.FindNodeExecution(workflow.ID, childExecution1.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeExecutionStateFinished, updatedChildExecution1.State)
	assert.Equal(t, models.WorkflowNodeExecutionResultCancelled, updatedChildExecution1.Result)

	updatedChildExecution2, err := models.FindNodeExecution(workflow.ID, childExecution2.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeExecutionStateFinished, updatedChildExecution2.State)
	assert.Equal(t, models.WorkflowNodeExecutionResultCancelled, updatedChildExecution2.Result)
}
