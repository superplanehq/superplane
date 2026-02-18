package workers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__CanvasNodeCleanupWorker_ProcessesSoftDeletedNode(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewCanvasNodeCleanupWorker()

	//
	// Create a canvas with two nodes
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{
				NodeID: "node-2",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	// Create associated data for both nodes
	event1 := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	event2 := support.EmitCanvasEventForNode(t, canvas.ID, "node-2", "default", nil)

	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event1.ID, event1.ID, nil)
	support.CreateCanvasNodeExecution(t, canvas.ID, "node-2", event2.ID, event2.ID, nil)

	support.CreateQueueItem(t, canvas.ID, "node-1", event1.ID, event1.ID)
	support.CreateQueueItem(t, canvas.ID, "node-2", event2.ID, event2.ID)

	// Create node request for node-1
	nodeRequest := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     "node-1",
		Type:       models.NodeRequestTypeInvokeAction,
		State:      models.NodeExecutionRequestStatePending,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "test",
				Parameters: map[string]any{},
			},
		}),
	}
	require.NoError(t, database.Conn().Create(&nodeRequest).Error)

	// Create execution KV for node-1
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(
		database.Conn(),
		canvas.ID,
		"node-1",
		execution1.ID,
		"test-key",
		"test-value",
	))

	//
	// Verify all data exists before soft delete
	//
	nodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
	support.VerifyCanvasEventsCount(t, canvas.ID, 2)
	support.VerifyNodeExecutionsCount(t, canvas.ID, 2)
	support.VerifyNodeQueueCount(t, canvas.ID, 2)

	//
	// Soft delete only node-1 (simulating a workflow update that removes one node)
	//
	err = database.Conn().Delete(&models.CanvasNode{}, "workflow_id = ? AND node_id = ?", canvas.ID, "node-1").Error
	require.NoError(t, err)

	// Verify node-1 is soft deleted (not visible in scoped queries)
	nodes, err = models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 1)
	assert.Equal(t, "node-2", nodes[0].NodeID)

	//
	// Process the deleted node with cleanup worker
	// May take multiple calls for batched cleanup
	//
	deletedNodes, err := models.ListDeletedCanvasNodes()
	require.NoError(t, err)
	require.Len(t, deletedNodes, 1)

	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		err = worker.LockAndProcessNode(deletedNodes[0])
		require.NoError(t, err)

		// Check if node is completely deleted
		var nodeCount int64
		database.Conn().Unscoped().Model(&models.CanvasNode{}).
			Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").
			Count(&nodeCount)
		if nodeCount == 0 {
			break
		}
	}

	//
	// Verify node-1 is permanently deleted with all its resources
	//
	var nodeCount int64
	database.Conn().Unscoped().Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").
		Count(&nodeCount)
	assert.Equal(t, int64(0), nodeCount)

	// Verify node-1's resources are deleted
	var eventCount int64
	database.Conn().Model(&models.CanvasEvent{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").
		Count(&eventCount)
	assert.Equal(t, int64(0), eventCount)

	var execCount int64
	database.Conn().Model(&models.CanvasNodeExecution{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").
		Count(&execCount)
	assert.Equal(t, int64(0), execCount)

	var queueCount int64
	database.Conn().Model(&models.CanvasNodeQueueItem{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").
		Count(&queueCount)
	assert.Equal(t, int64(0), queueCount)

	var requestCount int64
	database.Conn().Model(&models.CanvasNodeRequest{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").
		Count(&requestCount)
	assert.Equal(t, int64(0), requestCount)

	var kvCount int64
	database.Conn().Model(&models.CanvasNodeExecutionKV{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").
		Count(&kvCount)
	assert.Equal(t, int64(0), kvCount)

	//
	// Verify node-2 and its resources are untouched
	//
	nodes, err = models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 1)
	assert.Equal(t, "node-2", nodes[0].NodeID)

	var event2Count int64
	database.Conn().Model(&models.CanvasEvent{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-2").
		Count(&event2Count)
	assert.Equal(t, int64(1), event2Count)

	var exec2Count int64
	database.Conn().Model(&models.CanvasNodeExecution{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-2").
		Count(&exec2Count)
	assert.Equal(t, int64(1), exec2Count)
}

func Test__CanvasNodeCleanupWorker_IgnoresNonDeletedNodes(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewCanvasNodeCleanupWorker()

	//
	// Create a canvas with a node
	//
	canvas, createdNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	_ = support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)

	//
	// Try to process a non-deleted node (should be harmless)
	//
	err := worker.LockAndProcessNode(createdNodes[0])
	require.NoError(t, err)

	//
	// Verify node and data still exist
	//
	nodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 1)

	support.VerifyCanvasEventsCount(t, canvas.ID, 1)
}

func Test__CanvasNodeCleanupWorker_HandlesNodeWithNoResources(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewCanvasNodeCleanupWorker()

	//
	// Create a canvas with a node, but no associated resources
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Soft delete the node
	//
	err := database.Conn().Delete(&models.CanvasNode{}, "workflow_id = ? AND node_id = ?", canvas.ID, "node-1").Error
	require.NoError(t, err)

	//
	// Process the deleted node
	//
	deletedNodes, err := models.ListDeletedCanvasNodes()
	require.NoError(t, err)
	require.Len(t, deletedNodes, 1)

	err = worker.LockAndProcessNode(deletedNodes[0])
	require.NoError(t, err)

	//
	// Verify node is permanently deleted
	//
	var nodeCount int64
	database.Conn().Unscoped().Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").
		Count(&nodeCount)
	assert.Equal(t, int64(0), nodeCount)
}

func Test__CanvasNodeCleanupWorker_SkipsNodesFromDeletedCanvases(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	//
	// Create a canvas with a node
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Soft delete the node
	//
	err := database.Conn().Delete(&models.CanvasNode{}, "workflow_id = ? AND node_id = ?", canvas.ID, "node-1").Error
	require.NoError(t, err)

	//
	// Also soft delete the canvas itself
	//
	err = canvas.SoftDelete()
	require.NoError(t, err)

	//
	// ListDeletedCanvasNodes should NOT return this node
	// (it should be handled by the CanvasCleanupWorker instead)
	//
	deletedNodes, err := models.ListDeletedCanvasNodes()
	require.NoError(t, err)
	assert.Len(t, deletedNodes, 0)
}
