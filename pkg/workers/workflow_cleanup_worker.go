package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/telemetry"
)

type WorkflowCleanupWorker struct {
	semaphore           *semaphore.Weighted
	logger              *log.Entry
	maxResourcesPerTick int
}

func NewWorkflowCleanupWorker() *WorkflowCleanupWorker {
	return &WorkflowCleanupWorker{
		semaphore:           semaphore.NewWeighted(25),
		logger:              log.WithFields(log.Fields{"worker": "WorkflowCleanupWorker"}),
		maxResourcesPerTick: 500,
	}
}

func (w *WorkflowCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()
			canvases, err := models.ListDeletedCanvases()
			if err != nil {
				w.logger.Errorf("Error finding deleted workflows: %v", err)
				continue
			}

			telemetry.RecordWorkflowCleanupWorkerCanvasesCount(context.Background(), len(canvases))

			for _, canvas := range canvases {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(canvas models.Canvas) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessCanvas(canvas); err != nil {
						w.logger.Errorf("Error processing canvas %s: %v", canvas.ID, err)
					}
				}(canvas)
			}

			telemetry.RecordWorkflowCleanupWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *WorkflowCleanupWorker) LockAndProcessCanvas(canvas models.Canvas) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedCanvas, err := models.LockCanvas(tx, canvas.ID)
		if err != nil {
			w.logger.Infof("Canvas %s already being processed - skipping", canvas.ID)
			return nil
		}

		w.logger.Infof("Processing deleted canvas %s", lockedCanvas.ID)
		return w.processCanvas(tx, *lockedCanvas)
	})
}

func (w *WorkflowCleanupWorker) processCanvas(tx *gorm.DB, canvas models.Canvas) error {
	if !canvas.DeletedAt.Valid {
		w.logger.Infof("Skipping non-deleted canvas %s", canvas.ID)
		return nil
	}

	var nodes []models.CanvasNode
	err := tx.Unscoped().Where("workflow_id = ?", canvas.ID).Find(&nodes).Error
	if err != nil {
		return fmt.Errorf("failed to find workflow nodes: %w", err)
	}

	totalResourcesDeleted := 0
	nodesProcessed := 0

	for _, node := range nodes {
		if totalResourcesDeleted >= w.maxResourcesPerTick {
			w.logger.Infof("Reached max resources per tick (%d), stopping for this cycle", w.maxResourcesPerTick)
			break
		}

		resourcesDeleted, allResourcesDeleted, err := w.deleteNodeResourcesBatched(tx, canvas.ID, node.NodeID, w.maxResourcesPerTick-totalResourcesDeleted)
		if err != nil {
			return fmt.Errorf("failed to delete resources for node %s: %w", node.NodeID, err)
		}

		totalResourcesDeleted += resourcesDeleted

		if !allResourcesDeleted {
			w.logger.Infof("Partially cleaned node %s from canvas %s (deleted %d resources, more remain)", node.NodeID, canvas.ID, resourcesDeleted)
			nodesProcessed++

			continue
		}

		if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", canvas.ID, node.NodeID).Delete(&models.CanvasNode{}).Error; err != nil {
			return fmt.Errorf("failed to delete canvas node %s: %w", node.NodeID, err)
		}

		w.logger.Infof("Deleted node %s from canvas %s (deleted %d resources)", node.NodeID, canvas.ID, resourcesDeleted)
		nodesProcessed++
	}

	//
	// Check if all nodes are gone, then delete the canvas
	//
	var remainingNodesCount int64
	err = tx.Unscoped().Model(&models.CanvasNode{}).Where("workflow_id = ?", canvas.ID).Count(&remainingNodesCount).Error
	if err != nil {
		return fmt.Errorf("failed to check remaining canvas nodes: %w", err)
	}

	if remainingNodesCount > 0 {
		w.logger.Infof("Processed %d nodes from canvas %s (deleted %d resources, %d nodes remaining)", nodesProcessed, canvas.ID, totalResourcesDeleted, remainingNodesCount)
		return nil
	}

	w.logger.Infof("Processed %d nodes from canvas %s (deleted %d resources, %d nodes remaining)", nodesProcessed, canvas.ID, totalResourcesDeleted, remainingNodesCount)
	if err := tx.Unscoped().Delete(&canvas).Error; err != nil {
		return fmt.Errorf("failed to delete canvas: %w", err)
	}

	w.logger.Infof("Successfully cleaned up canvas %s (deleted %d resources total)", canvas.ID, totalResourcesDeleted)
	return nil
}

func (w *WorkflowCleanupWorker) deleteNodeResources(tx *gorm.DB, canvasID uuid.UUID, nodeID string) error {
	if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", canvasID, nodeID).Delete(&models.CanvasNodeRequest{}).Error; err != nil {
		return fmt.Errorf("failed to delete node requests: %w", err)
	}

	if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", canvasID, nodeID).Delete(&models.CanvasNodeExecutionKV{}).Error; err != nil {
		return fmt.Errorf("failed to delete node execution KVs: %w", err)
	}

	if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", canvasID, nodeID).Delete(&models.CanvasNodeExecution{}).Error; err != nil {
		return fmt.Errorf("failed to delete node executions: %w", err)
	}

	if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", canvasID, nodeID).Delete(&models.CanvasNodeQueueItem{}).Error; err != nil {
		return fmt.Errorf("failed to delete node queue items: %w", err)
	}

	if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", canvasID, nodeID).Delete(&models.CanvasEvent{}).Error; err != nil {
		return fmt.Errorf("failed to delete workflow events: %w", err)
	}

	return nil
}

func (w *WorkflowCleanupWorker) deleteNodeResourcesBatched(tx *gorm.DB, workflowID uuid.UUID, nodeID string, maxResources int) (resourcesDeleted int, allResourcesDeleted bool, err error) {
	resourceTypes := []struct {
		model     interface{}
		tableName string
	}{
		{&models.CanvasNodeRequest{}, "canvas_node_requests"},
		{&models.CanvasNodeExecutionKV{}, "canvas_node_execution_kvs"},
		{&models.CanvasNodeExecution{}, "canvas_node_executions"},
		{&models.CanvasNodeQueueItem{}, "canvas_node_queue_items"},
		{&models.CanvasEvent{}, "canvas_events"},
	}

	totalDeleted := 0
	allDeleted := true

	for _, resourceType := range resourceTypes {
		if totalDeleted >= maxResources {
			allDeleted = false
			break
		}

		remaining := maxResources - totalDeleted

		// Delete in batches with LIMIT
		result := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Limit(remaining).Delete(resourceType.model)
		if result.Error != nil {
			return totalDeleted, false, fmt.Errorf("failed to delete %s: %w", resourceType.tableName, result.Error)
		}

		deleted := int(result.RowsAffected)
		totalDeleted += deleted

		if deleted != remaining {
			continue
		}

		var count int64

		if err := tx.Unscoped().Model(resourceType.model).Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Count(&count).Error; err != nil {
			return totalDeleted, false, fmt.Errorf("failed to count remaining %s: %w", resourceType.tableName, err)
		}

		if count > 0 {
			allDeleted = false
			break
		}
	}

	return totalDeleted, allDeleted, nil
}
