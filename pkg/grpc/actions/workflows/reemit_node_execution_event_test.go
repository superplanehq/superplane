package workflows

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func Test_ReEmitNodeExecutionEvent_Success(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	nodeID := "test-node"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: nodeID,
				Name:   "Test Node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, nodeID, "default", nil)
	event := support.EmitWorkflowEventForNode(t, workflow.ID, nodeID, "default", nil)

	nodeExecution := support.CreateWorkflowNodeExecution(t, workflow.ID, nodeID, rootEvent.ID, event.ID, nil)

	initialItems, err := models.ListNodeQueueItems(workflow.ID, nodeID, 100, nil)
	require.NoError(t, err)
	initialCount := len(initialItems)

	response, err := ReEmitNodeExecutionEvent(
		context.Background(),
		r.Organization.ID,
		workflow.ID,
		nodeID,
		nodeExecution.ID,
	)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, rootEvent.ID.String(), response.EventId)

	finalItems, err := models.ListNodeQueueItems(workflow.ID, nodeID, 100, nil)
	require.NoError(t, err)
	assert.Equal(t, initialCount+1, len(finalItems))

	var newItem *models.WorkflowNodeQueueItem
	for _, item := range finalItems {
		found := false
		for _, initialItem := range initialItems {
			if item.ID == initialItem.ID {
				found = true
				break
			}
		}
		if !found {
			newItem = &item
			break
		}
	}

	require.NotNil(t, newItem)
	assert.Equal(t, workflow.ID, newItem.WorkflowID)
	assert.Equal(t, nodeID, newItem.NodeID)
	assert.Equal(t, rootEvent.ID, newItem.RootEventID)
	assert.Equal(t, event.ID, newItem.EventID)
	assert.NotNil(t, newItem.CreatedAt)
}

func Test_ReEmitNodeExecutionEvent_WorkflowNotFound(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	nonExistentWorkflowID := uuid.New()
	nonExistentExecutionID := uuid.New()

	response, err := ReEmitNodeExecutionEvent(
		context.Background(),
		r.Organization.ID,
		nonExistentWorkflowID,
		"test-node",
		nonExistentExecutionID,
	)

	require.Error(t, err)
	assert.Nil(t, response)

	grpcErr, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, grpcErr.Code())
	assert.Equal(t, "workflow not found", grpcErr.Message())
}

func Test_ReEmitNodeExecutionEvent_NodeNotFound(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "existing-node",
				Name:   "Existing Node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	nonExistentExecutionID := uuid.New()

	response, err := ReEmitNodeExecutionEvent(
		context.Background(),
		r.Organization.ID,
		workflow.ID,
		"non-existent-node",
		nonExistentExecutionID,
	)

	require.Error(t, err)
	assert.Nil(t, response)

	grpcErr, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, grpcErr.Code())
	assert.Equal(t, "node not found", grpcErr.Message())
}

func Test_ReEmitNodeExecutionEvent_NodeExecutionNotFound(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	nodeID := "test-node"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: nodeID,
				Name:   "Test Node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	nonExistentExecutionID := uuid.New()

	response, err := ReEmitNodeExecutionEvent(
		context.Background(),
		r.Organization.ID,
		workflow.ID,
		nodeID,
		nonExistentExecutionID,
	)

	require.Error(t, err)
	assert.Nil(t, response)

	grpcErr, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, grpcErr.Code())
	assert.Equal(t, "node execution not found", grpcErr.Message())
}

func Test_ReEmitNodeExecutionEvent_MultipleReemits(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	nodeID := "test-node"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: nodeID,
				Name:   "Test Node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, nodeID, "default", nil)
	event := support.EmitWorkflowEventForNode(t, workflow.ID, nodeID, "default", nil)

	nodeExecution := support.CreateWorkflowNodeExecution(t, workflow.ID, nodeID, rootEvent.ID, event.ID, nil)

	response1, err := ReEmitNodeExecutionEvent(
		context.Background(),
		r.Organization.ID,
		workflow.ID,
		nodeID,
		nodeExecution.ID,
	)
	require.NoError(t, err)
	require.NotNil(t, response1)
	assert.Equal(t, rootEvent.ID.String(), response1.EventId)

	response2, err := ReEmitNodeExecutionEvent(
		context.Background(),
		r.Organization.ID,
		workflow.ID,
		nodeID,
		nodeExecution.ID,
	)
	require.NoError(t, err)
	require.NotNil(t, response2)
	assert.Equal(t, rootEvent.ID.String(), response2.EventId)

	items, err := models.ListNodeQueueItems(workflow.ID, nodeID, 100, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, len(items))

	for _, item := range items {
		assert.Equal(t, workflow.ID, item.WorkflowID)
		assert.Equal(t, nodeID, item.NodeID)
		assert.Equal(t, rootEvent.ID, item.RootEventID)
		assert.Equal(t, event.ID, item.EventID)
	}
}
