package workers

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
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

func Test__CanvasNodeCleanupWorker_RotatesBlockedNodesSoOthersProgress(t *testing.T) {
	r := support.Setup(t)
	worker := NewCanvasNodeCleanupWorker()
	worker.maxNodesPerTick = 1
	worker.pauseBetweenBatches = 0

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "blocked-node",
				Type:   models.NodeTypeTrigger,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Trigger: &models.TriggerRef{Name: "noop"},
				}),
			},
			{
				NodeID: "keep-node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{
				NodeID: "ready-node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "blocked-node", "default", nil)
	support.CreateCanvasNodeExecution(t, canvas.ID, "keep-node", rootEvent.ID, rootEvent.ID)
	support.EmitCanvasEventForNode(t, canvas.ID, "ready-node", "default", nil)

	older := time.Now().AddDate(0, 0, -40)
	newer := time.Now().AddDate(0, 0, -35)
	blocked := softDeleteCanvasNode(t, canvas.ID, "blocked-node", older)
	ready := softDeleteCanvasNode(t, canvas.ID, "ready-node", newer)

	// Force blocked node to the front of the cleanup queue.
	require.NoError(t, database.Conn().Unscoped().Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "blocked-node").
		Update("updated_at", older).Error)
	require.NoError(t, database.Conn().Unscoped().Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "ready-node").
		Update("updated_at", newer).Error)

	candidates, err := models.ListDeletedCanvasNodes(database.Conn(), time.Now().AddDate(0, 0, -deletedResourceGracePeriodDays), 1)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, "blocked-node", candidates[0].NodeID)

	require.NoError(t, worker.LockAndProcessNode(blocked))
	assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "blocked-node"))

	candidates, err = models.ListDeletedCanvasNodes(database.Conn(), time.Now().AddDate(0, 0, -deletedResourceGracePeriodDays), 1)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, "ready-node", candidates[0].NodeID)

	require.NoError(t, worker.LockAndProcessNode(ready))
	assert.Equal(t, int64(0), countUnscopedCanvasNodes(t, canvas.ID, "ready-node"))
	assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "blocked-node"))
}

func Test__CanvasNodeCleanupWorker_PreservesEventsReferencedByOtherNodes(t *testing.T) {
	r := support.Setup(t)
	worker := NewCanvasNodeCleanupWorker()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "source-node",
				Type:   models.NodeTypeTrigger,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Trigger: &models.TriggerRef{Name: "noop"},
				}),
			},
			{
				NodeID: "keep-node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "source-node", "default", nil)
	keepExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "keep-node", rootEvent.ID, rootEvent.ID)

	deletedNode := softDeleteCanvasNode(t, canvas.ID, "source-node", time.Now().AddDate(0, 0, -31))
	require.NoError(t, worker.LockAndProcessNode(deletedNode))

	assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "source-node"))
	assert.Equal(t, int64(1), countNodeEvents(t, canvas.ID, "source-node"))
	assert.Equal(t, int64(1), countNodeExecutions(t, canvas.ID, "keep-node"))

	var keepExecutionAfter models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", keepExecution.ID).First(&keepExecutionAfter).Error)
	assert.Equal(t, rootEvent.ID, keepExecutionAfter.RootEventID)
	assert.Equal(t, rootEvent.ID, keepExecutionAfter.EventID)

	require.NoError(t, database.Conn().Where("id = ?", keepExecution.ID).Delete(&models.CanvasNodeExecution{}).Error)
	require.NoError(t, worker.LockAndProcessNode(deletedNode))

	assert.Equal(t, int64(0), countUnscopedCanvasNodes(t, canvas.ID, "source-node"))
	assert.Equal(t, int64(0), countNodeEvents(t, canvas.ID, "source-node"))
	assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "keep-node"))
}

func Test__CanvasNodeCleanupWorker_DoesNotCascadeDeleteReferencedOutputEvents(t *testing.T) {
	r := support.Setup(t)
	worker := NewCanvasNodeCleanupWorker()
	worker.pauseBetweenBatches = 0

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "source-node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{
				NodeID: "keep-node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	inputEvent := support.EmitCanvasEventForNode(t, canvas.ID, "source-node", "default", nil)
	sourceExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "source-node", inputEvent.ID, inputEvent.ID)
	outputEvent := support.EmitCanvasEventForNode(t, canvas.ID, "source-node", "default", &sourceExecution.ID)
	keepExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "keep-node", outputEvent.ID, outputEvent.ID)

	deletedNode := softDeleteCanvasNode(t, canvas.ID, "source-node", time.Now().AddDate(0, 0, -31))
	require.NoError(t, worker.LockAndProcessNode(deletedNode))

	assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "source-node"))
	assert.Equal(t, int64(1), countNodeExecutions(t, canvas.ID, "source-node"))

	var outputAfter models.CanvasEvent
	require.NoError(t, database.Conn().Where("id = ?", outputEvent.ID).First(&outputAfter).Error)
	assert.Equal(t, sourceExecution.ID, *outputAfter.ExecutionID)

	var keepExecutionAfter models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", keepExecution.ID).First(&keepExecutionAfter).Error)
	assert.Equal(t, outputEvent.ID, keepExecutionAfter.RootEventID)
	assert.Equal(t, outputEvent.ID, keepExecutionAfter.EventID)
}

func Test__CanvasNodeCleanupWorker_HandlesMultiTickBatching(t *testing.T) {
	r := support.Setup(t)
	worker := NewCanvasNodeCleanupWorker()
	worker.maxResourcesPerNodePerTick = 2
	worker.deleteBatchSize = 2
	worker.pauseBetweenBatches = 0

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

	deletedNode := softDeleteCanvasNode(t, canvas.ID, "node-1", time.Now().AddDate(0, 0, -31))
	require.NoError(t, canvas.SoftDelete())
	require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).
		Where("id = ?", canvas.ID).
		Update("deleted_at", time.Now().AddDate(0, 0, -31)).Error)

	nodes, err := models.ListDeletedCanvasNodes(database.Conn(), time.Now().AddDate(0, 0, -deletedResourceGracePeriodDays), 100)
	require.NoError(t, err)
	for _, node := range nodes {
		assert.NotEqual(t, canvas.ID, node.WorkflowID)
	}

	require.NoError(t, worker.LockAndProcessNode(deletedNode))

	assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "node-1"))
	assert.Equal(t, int64(1), countNodeEvents(t, canvas.ID, "node-1"))
	assert.Equal(t, int64(1), countNodeExecutions(t, canvas.ID, "node-1"))
}

func Test__CanvasNodeCleanupWorker_IgnoresNodesOnDeletedOrganization(t *testing.T) {
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

	deletedNode := softDeleteCanvasNode(t, canvas.ID, "node-1", time.Now().AddDate(0, 0, -31))
	require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))

	nodes, err := models.ListDeletedCanvasNodes(database.Conn(), time.Now().AddDate(0, 0, -deletedResourceGracePeriodDays), 100)
	require.NoError(t, err)
	for _, node := range nodes {
		assert.NotEqual(t, canvas.ID, node.WorkflowID)
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		_, err := models.LockDeletedCanvasNode(tx, canvas.ID, "node-1")
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		return nil
	})
	require.NoError(t, err)

	require.NoError(t, worker.LockAndProcessNode(deletedNode))

	assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "node-1"))
	assert.Equal(t, int64(1), countNodeEvents(t, canvas.ID, "node-1"))
	assert.Equal(t, int64(1), countNodeExecutions(t, canvas.ID, "node-1"))
}

func Test__ListDeletedCanvasNodes_ExcludesNodesWithinGracePeriod(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "in-grace",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{
				NodeID: "past-grace",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	_ = softDeleteCanvasNode(t, canvas.ID, "in-grace", time.Now().AddDate(0, 0, -10))
	_ = softDeleteCanvasNode(t, canvas.ID, "past-grace", time.Now().AddDate(0, 0, -40))

	eligibleBefore := time.Now().AddDate(0, 0, -deletedResourceGracePeriodDays)
	nodes, err := models.ListDeletedCanvasNodes(database.Conn(), eligibleBefore, 100)
	require.NoError(t, err)

	foundPastGrace := false
	for _, node := range nodes {
		if node.WorkflowID != canvas.ID {
			continue
		}
		assert.NotEqual(t, "in-grace", node.NodeID)
		if node.NodeID == "past-grace" {
			foundPastGrace = true
		}
	}
	require.True(t, foundPastGrace)
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

	nodes, err := models.ListDeletedCanvasNodes(database.Conn(), time.Now().AddDate(0, 0, -deletedResourceGracePeriodDays), 100)
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

	limited, err := models.ListDeletedCanvasNodes(database.Conn(), time.Now().AddDate(0, 0, -deletedResourceGracePeriodDays), 1)
	require.NoError(t, err)
	assert.Len(t, limited, 1)

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

func Test__CanvasNodeCleanupWorker_LoadDoesNotStarveHotPath(t *testing.T) {
	r := support.Setup(t)

	const (
		deletedNodeCount = 8
		requestsPerNode  = 400
	)

	nodes := []models.CanvasNode{
		{
			NodeID: "live-node",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "noop"},
			}),
		},
	}
	for i := 0; i < deletedNodeCount; i++ {
		nodes = append(nodes, models.CanvasNode{
			NodeID: uuid.NewString(),
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "noop"},
			}),
		})
	}

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nodes, []models.Edge{})
	liveEvent := support.EmitCanvasEventForNode(t, canvas.ID, "live-node", "default", nil)

	deletedNodes := make([]models.CanvasNode, 0, deletedNodeCount)
	for _, node := range nodes[1:] {
		seedNodeRequests(t, canvas.ID, node.NodeID, requestsPerNode)
		deletedNodes = append(deletedNodes, softDeleteCanvasNode(t, canvas.ID, node.NodeID, time.Now().AddDate(0, 0, -31)))
	}

	worker := NewCanvasNodeCleanupWorker()
	worker.pauseBetweenBatches = 0
	worker.maxNodesPerTick = deletedNodeCount

	var (
		latenciesMu sync.Mutex
		latencies   []time.Duration
		stopHotPath atomic.Bool
		hotPathErr  atomic.Value
	)

	hotPathDone := make(chan struct{})
	go func() {
		defer close(hotPathDone)
		for !stopHotPath.Load() {
			started := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			var event models.CanvasEvent
			err := database.Conn().WithContext(ctx).
				Where("id = ?", liveEvent.ID).
				First(&event).Error
			cancel()
			latenciesMu.Lock()
			latencies = append(latencies, time.Since(started))
			latenciesMu.Unlock()
			if err != nil {
				hotPathErr.Store(err)
				return
			}
		}
	}()

	for pass := 0; pass < 40; pass++ {
		remaining := 0
		for _, node := range deletedNodes {
			require.NoError(t, worker.LockAndProcessNode(node))
			if countUnscopedCanvasNodes(t, canvas.ID, node.NodeID) > 0 {
				remaining++
			}
		}
		if remaining == 0 {
			break
		}
	}

	stopHotPath.Store(true)
	<-hotPathDone

	if err, ok := hotPathErr.Load().(error); ok && err != nil {
		require.NoError(t, err)
	}

	for _, node := range deletedNodes {
		assert.Equal(t, int64(0), countUnscopedCanvasNodes(t, canvas.ID, node.NodeID))
		var requestCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.CanvasNodeRequest{}).
			Where("workflow_id = ? AND node_id = ?", canvas.ID, node.NodeID).
			Count(&requestCount).Error)
		assert.Equal(t, int64(0), requestCount)
	}

	assert.Equal(t, int64(1), countUnscopedCanvasNodes(t, canvas.ID, "live-node"))
	assert.LessOrEqual(t, worker.maxObservedInFlight.Load(), int32(1))

	latenciesMu.Lock()
	require.NotEmpty(t, latencies)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p95 := latencies[(len(latencies)*95)/100]
	latenciesMu.Unlock()

	assert.Less(t, p95, 500*time.Millisecond, "hot-path SELECT p95 should stay responsive during cleanup")
}

func seedNodeRequests(t *testing.T, workflowID uuid.UUID, nodeID string, count int) {
	t.Helper()

	now := time.Now()
	requests := make([]models.CanvasNodeRequest, 0, count)
	for i := 0; i < count; i++ {
		requests = append(requests, models.CanvasNodeRequest{
			ID:         uuid.New(),
			WorkflowID: workflowID,
			NodeID:     nodeID,
			State:      models.NodeExecutionRequestStatePending,
			Type:       models.NodeRequestTypeInvokeAction,
			Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
				InvokeAction: &models.InvokeAction{
					ActionName: "poll",
					Parameters: map[string]any{},
				},
			}),
			RunAt:     now.Add(time.Hour),
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	require.NoError(t, database.Conn().CreateInBatches(requests, 200).Error)
}
