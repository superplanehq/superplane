package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
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
			w.tick()
		}
	}
}

func (w *CanvasNodeCleanupWorker) tick() {
	tickStart := time.Now()
	nodes, err := models.ListDeletedCanvasNodes(database.Conn())
	if err != nil {
		w.logger.Errorf("Error finding deleted canvas nodes: %v", err)
		return
	}

	for _, node := range nodes {
		if !node.DeletedAt.Valid || deletedResourceWithinGracePeriod(node.DeletedAt.Time, tickStart) {
			continue
		}

		if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
			w.logger.Errorf("Error acquiring semaphore: %v", err)
			continue
		}

		go w.processDeletedNode(node)
	}

	w.logger.WithFields(log.Fields{
		"nodes":       len(nodes),
		"duration_ms": time.Since(tickStart).Milliseconds(),
	}).Debug("Canvas node cleanup tick completed")
}

func (w *CanvasNodeCleanupWorker) processDeletedNode(node models.CanvasNode) {
	defer w.semaphore.Release(1)

	if err := w.LockAndProcessNode(node); err != nil {
		w.logger.Errorf("Error processing canvas node %s/%s: %v", node.WorkflowID, node.NodeID, err)
	}
}

func (w *CanvasNodeCleanupWorker) LockAndProcessNode(node models.CanvasNode) error {
	if !node.DeletedAt.Valid || deletedResourceWithinGracePeriod(node.DeletedAt.Time, time.Now()) {
		return nil
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedNode, err := models.LockDeletedCanvasNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.logger.Infof("Canvas node %s/%s already being processed - skipping", node.WorkflowID, node.NodeID)
				return nil
			}

			return fmt.Errorf("lock deleted canvas node %s/%s: %w", node.WorkflowID, node.NodeID, err)
		}

		if !lockedNode.DeletedAt.Valid || deletedResourceWithinGracePeriod(lockedNode.DeletedAt.Time, time.Now()) {
			return nil
		}

		w.logger.Infof("Processing deleted canvas node %s/%s", lockedNode.WorkflowID, lockedNode.NodeID)
		return w.processNode(tx, *lockedNode)
	})
}

func (w *CanvasNodeCleanupWorker) processNode(tx *gorm.DB, node models.CanvasNode) error {
	result, err := models.NewNodeResourceCleaner(tx, &node).
		ForUnreferenced().
		WithLimit(w.maxResourcesPerTick).
		Run()
	if err != nil {
		return fmt.Errorf("failed to delete resources for node %s: %w", node.NodeID, err)
	}

	if !result.AllDeleted {
		w.logger.Infof(
			"Partially cleaned node %s from canvas %s (deleted %d resources, more remain)",
			node.NodeID,
			node.WorkflowID,
			result.ResourcesDeleted,
		)
		return nil
	}

	if err := node.HardDelete(tx); err != nil {
		return fmt.Errorf("failed to delete canvas node %s: %w", node.NodeID, err)
	}

	w.logger.Infof(
		"Deleted node %s from canvas %s (deleted %d resources)",
		node.NodeID,
		node.WorkflowID,
		result.ResourcesDeleted,
	)
	return nil
}
