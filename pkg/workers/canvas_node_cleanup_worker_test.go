package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func softDeleteCanvasNode(t *testing.T, canvasID uuid.UUID, nodeID string, deletedAt time.Time) models.CanvasNode {
	t.Helper()

	node, err := models.FindCanvasNode(database.Conn(), canvasID, nodeID)
	require.NoError(t, err)
	require.NoError(t, models.DeleteCanvasNode(database.Conn(), *node))

	require.NoError(t, database.Conn().Unscoped().Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvasID, nodeID).
		Update("deleted_at", deletedAt).Error)

	deletedNode, err := models.FindUnscopedCanvasNode(database.Conn(), canvasID, nodeID)
	require.NoError(t, err)
	require.True(t, deletedNode.DeletedAt.Valid)
	return *deletedNode
}

func countUnscopedCanvasNodes(t *testing.T, workflowID uuid.UUID, nodeID string) int64 {
	t.Helper()

	var count int64
	require.NoError(t, database.Conn().Unscoped().Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).
		Count(&count).Error)
	return count
}

func countNodeEvents(t *testing.T, workflowID uuid.UUID, nodeID string) int64 {
	t.Helper()

	var count int64
	require.NoError(t, database.Conn().Unscoped().Model(&models.CanvasEvent{}).
		Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).
		Count(&count).Error)
	return count
}

func countNodeExecutions(t *testing.T, workflowID uuid.UUID, nodeID string) int64 {
	t.Helper()

	var count int64
	require.NoError(t, database.Conn().Unscoped().Model(&models.CanvasNodeExecution{}).
		Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).
		Count(&count).Error)
	return count
}

func Test__CanvasNodeCleanupWorker_GracePeriod(t *testing.T) {
	r := support.Setup(t)

	t.Run("skips cleanup while node is still within grace period", func(t *testing.T) {
		worker := NewCanvasNodeCleanupWorker()
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

		event := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
		support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event.ID, event.ID)

		deletedNode := softDeleteCanvasNode(t, canvas.ID, "node-1", time.Now().AddDate(0, 0, -29))
		require.NoError(t, worker.LockAndProcessNode(deletedNode))

		assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "node-1"))
		assert.Equal(t, int64(1), countNodeEvents(t, canvas.ID, "node-1"))
		assert.Equal(t, int64(1), countNodeExecutions(t, canvas.ID, "node-1"))
	})

	t.Run("cleans up node after grace period expires", func(t *testing.T) {
		worker := NewCanvasNodeCleanupWorker()
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

		event := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event.ID, event.ID)
		require.NoError(t, models.CreateNodeExecutionKVInTransaction(
			database.Conn(),
			canvas.ID,
			"node-1",
			execution.ID,
			"test-key",
			"test-value",
		))

		deletedNode := softDeleteCanvasNode(t, canvas.ID, "node-1", time.Now().AddDate(0, 0, -31))
		require.NoError(t, worker.LockAndProcessNode(deletedNode))

		assert.Equal(t, int64(0), countUnscopedCanvasNodes(t, canvas.ID, "node-1"))
		assert.Equal(t, int64(0), countNodeEvents(t, canvas.ID, "node-1"))
		assert.Equal(t, int64(0), countNodeExecutions(t, canvas.ID, "node-1"))

		var kvCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.CanvasNodeExecutionKV{}).
			Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").
			Count(&kvCount).Error)
		assert.Equal(t, int64(0), kvCount)

		_, err := models.FindCanvas(r.Organization.ID, canvas.ID)
		require.NoError(t, err)
	})
}

func Test__CanvasNodeCleanupWorker_ProcessesDeletedNodeResources(t *testing.T) {
	r := support.Setup(t)
	worker := NewCanvasNodeCleanupWorker()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "keep-node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{
				NodeID: "delete-node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	keepEvent := support.EmitCanvasEventForNode(t, canvas.ID, "keep-node", "default", nil)
	support.CreateCanvasNodeExecution(t, canvas.ID, "keep-node", keepEvent.ID, keepEvent.ID)

	deleteEvent := support.EmitCanvasEventForNode(t, canvas.ID, "delete-node", "default", nil)
	support.CreateCanvasNodeExecution(t, canvas.ID, "delete-node", deleteEvent.ID, deleteEvent.ID)

	deletedNode := softDeleteCanvasNode(t, canvas.ID, "delete-node", time.Now().AddDate(0, 0, -31))
	require.NoError(t, worker.LockAndProcessNode(deletedNode))

	assert.Equal(t, int64(0), countUnscopedCanvasNodes(t, canvas.ID, "delete-node"))
	assert.Equal(t, int64(0), countNodeEvents(t, canvas.ID, "delete-node"))
	assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "keep-node"))
	assert.Equal(t, int64(1), countNodeEvents(t, canvas.ID, "keep-node"))
	assert.Equal(t, int64(1), countNodeExecutions(t, canvas.ID, "keep-node"))
}

func Test__CanvasNodeCleanupWorker_HandlesMultiTickBatching(t *testing.T) {
	r := support.Setup(t)
	worker := NewCanvasNodeCleanupWorker()
	worker.maxResourcesPerTick = 2

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

	for i := 0; i < 5; i++ {
		support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	}

	deletedNode := softDeleteCanvasNode(t, canvas.ID, "node-1", time.Now().AddDate(0, 0, -31))

	require.NoError(t, worker.LockAndProcessNode(deletedNode))
	assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "node-1"))
	assert.Equal(t, int64(3), countNodeEvents(t, canvas.ID, "node-1"))

	require.NoError(t, worker.LockAndProcessNode(deletedNode))
	assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "node-1"))
	assert.Equal(t, int64(1), countNodeEvents(t, canvas.ID, "node-1"))

	require.NoError(t, worker.LockAndProcessNode(deletedNode))
	assert.Equal(t, int64(0), countUnscopedCanvasNodes(t, canvas.ID, "node-1"))
	assert.Equal(t, int64(0), countNodeEvents(t, canvas.ID, "node-1"))
}

func Test__CanvasNodeCleanupWorker_HandlesConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)

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

	event := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event.ID, event.ID)

	deletedNode := softDeleteCanvasNode(t, canvas.ID, "node-1", time.Now().AddDate(0, 0, -31))

	results := make(chan error, 2)
	go func() {
		results <- NewCanvasNodeCleanupWorker().LockAndProcessNode(deletedNode)
	}()
	go func() {
		results <- NewCanvasNodeCleanupWorker().LockAndProcessNode(deletedNode)
	}()

	assert.NoError(t, <-results)
	assert.NoError(t, <-results)

	for i := 0; i < 5; i++ {
		require.NoError(t, NewCanvasNodeCleanupWorker().LockAndProcessNode(deletedNode))
		if countUnscopedCanvasNodes(t, canvas.ID, "node-1") == 0 {
			break
		}
	}

	assert.Equal(t, int64(0), countUnscopedCanvasNodes(t, canvas.ID, "node-1"))
	assert.Equal(t, int64(0), countNodeEvents(t, canvas.ID, "node-1"))
	assert.Equal(t, int64(0), countNodeExecutions(t, canvas.ID, "node-1"))
}

func Test__CanvasNodeCleanupWorker_IgnoresNonDeletedNodes(t *testing.T) {
	r := support.Setup(t)
	worker := NewCanvasNodeCleanupWorker()

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

	event := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event.ID, event.ID)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, "node-1")
	require.NoError(t, err)

	require.NoError(t, worker.LockAndProcessNode(*node))

	assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "node-1"))
	assert.Equal(t, int64(1), countNodeEvents(t, canvas.ID, "node-1"))
	assert.Equal(t, int64(1), countNodeExecutions(t, canvas.ID, "node-1"))
}

func Test__CanvasNodeCleanupWorker_IgnoresNodesOnDeletedCanvas(t *testing.T) {
	r := support.Setup(t)

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

	event := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event.ID, event.ID)

	_ = softDeleteCanvasNode(t, canvas.ID, "node-1", time.Now().AddDate(0, 0, -31))
	require.NoError(t, canvas.SoftDelete())
	require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).
		Where("id = ?", canvas.ID).
		Update("deleted_at", time.Now().AddDate(0, 0, -31)).Error)

	nodes, err := models.ListDeletedCanvasNodes(database.Conn())
	require.NoError(t, err)
	for _, node := range nodes {
		assert.NotEqual(t, canvas.ID, node.WorkflowID)
	}

	assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "node-1"))
	assert.Equal(t, int64(1), countNodeEvents(t, canvas.ID, "node-1"))
}

func Test__ListDeletedCanvasNodes_AndLock(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "live-node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{
				NodeID: "deleted-node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	_ = softDeleteCanvasNode(t, canvas.ID, "deleted-node", time.Now().AddDate(0, 0, -31))

	nodes, err := models.ListDeletedCanvasNodes(database.Conn())
	require.NoError(t, err)

	found := false
	for _, node := range nodes {
		if node.WorkflowID == canvas.ID && node.NodeID == "deleted-node" {
			found = true
		}
		if node.WorkflowID == canvas.ID {
			assert.NotEqual(t, "live-node", node.NodeID)
		}
	}
	require.True(t, found)

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockDeletedCanvasNode(tx, canvas.ID, "deleted-node")
		require.NoError(t, err)
		assert.Equal(t, "deleted-node", locked.NodeID)
		assert.True(t, locked.DeletedAt.Valid)

		_, err = models.LockDeletedCanvasNode(tx, canvas.ID, "live-node")
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		return nil
	})
	require.NoError(t, err)
}
