package workers

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/telemetry"
)

type CanvasNodeCleanupWorker struct {
	semaphore           *semaphore.Weighted
	logger              *log.Entry
	maxResourcesPerTick int
}

func NewCanvasNodeCleanupWorker() *CanvasNodeCleanupWorker {
	return &CanvasNodeCleanupWorker{
		semaphore:           semaphore.NewWeighted(25),
		logger:              log.WithFields(log.Fields{"worker": "CanvasNodeCleanupWorker"}),
		maxResourcesPerTick: 500,
	}
}

func (w *CanvasNodeCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()
			nodes, err := models.ListDeletedCanvasNodes()
			if err != nil {
				w.logger.Errorf("Error finding deleted nodes: %v", err)
				continue
			}

			telemetry.RecordNodeCleanupWorkerNodesCount(context.Background(), len(nodes))

			for _, node := range nodes {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(node models.CanvasNode) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessNode(node); err != nil {
						w.logger.Errorf("Error processing node %s from canvas %s: %v", node.NodeID, node.WorkflowID, err)
					}
				}(node)
			}

			telemetry.RecordNodeCleanupWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *CanvasNodeCleanupWorker) LockAndProcessNode(node models.CanvasNode) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedNode, err := models.LockDeletedCanvasNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			w.logger.Infof("Node %s from canvas %s already being processed - skipping", node.NodeID, node.WorkflowID)
			return nil
		}

		w.logger.Infof("Processing deleted node %s from canvas %s", lockedNode.NodeID, lockedNode.WorkflowID)
		return w.processNode(tx, *lockedNode)
	})
}

func (w *CanvasNodeCleanupWorker) processNode(tx *gorm.DB, node models.CanvasNode) error {
	if !node.DeletedAt.Valid {
		w.logger.Infof("Skipping non-deleted node %s from canvas %s", node.NodeID, node.WorkflowID)
		return nil
	}

	resourcesDeleted, allResourcesDeleted, err := deleteNodeResourcesBatched(tx, node.WorkflowID, node.NodeID, w.maxResourcesPerTick)
	if err != nil {
		return fmt.Errorf("failed to delete resources for node %s: %w", node.NodeID, err)
	}

	if !allResourcesDeleted {
		w.logger.Infof("Partially cleaned node %s from canvas %s (deleted %d resources, more remain)", node.NodeID, node.WorkflowID, resourcesDeleted)
		return nil
	}

	if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", node.WorkflowID, node.NodeID).Delete(&models.CanvasNode{}).Error; err != nil {
		return fmt.Errorf("failed to delete canvas node %s: %w", node.NodeID, err)
	}

	w.logger.Infof("Successfully cleaned up node %s from canvas %s (deleted %d resources)", node.NodeID, node.WorkflowID, resourcesDeleted)
	return nil
}
