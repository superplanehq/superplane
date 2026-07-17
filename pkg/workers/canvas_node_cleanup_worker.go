package workers

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	canvasNodeCleanupTickEvery           = 30 * time.Second
	canvasNodeCleanupMaxNodesPerTick     = 10
	canvasNodeCleanupDeleteBatchSize     = 50
	canvasNodeCleanupMaxResourcesPerNode = 200
	canvasNodeCleanupPauseBetweenBatches = 50 * time.Millisecond
)

type CanvasNodeCleanupWorker struct {
	logger                     *log.Entry
	maxNodesPerTick            int
	deleteBatchSize            int
	maxResourcesPerNodePerTick int
	pauseBetweenBatches        time.Duration
	inFlightBatches            *atomic.Int32
	maxObservedInFlight        *atomic.Int32
}

func NewCanvasNodeCleanupWorker() *CanvasNodeCleanupWorker {
	return &CanvasNodeCleanupWorker{
		logger:                     log.WithFields(log.Fields{"worker": "CanvasNodeCleanupWorker"}),
		maxNodesPerTick:            canvasNodeCleanupMaxNodesPerTick,
		deleteBatchSize:            canvasNodeCleanupDeleteBatchSize,
		maxResourcesPerNodePerTick: canvasNodeCleanupMaxResourcesPerNode,
		pauseBetweenBatches:        canvasNodeCleanupPauseBetweenBatches,
		inFlightBatches:            &atomic.Int32{},
		maxObservedInFlight:        &atomic.Int32{},
	}
}

func (w *CanvasNodeCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(canvasNodeCleanupTickEvery)
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
	nodes, err := models.ListDeletedCanvasNodes(database.Conn(), w.maxNodesPerTick)
	if err != nil {
		w.logger.Errorf("Error finding deleted canvas nodes: %v", err)
		return
	}

	processed := 0
	for _, node := range nodes {
		if !node.DeletedAt.Valid || deletedResourceWithinGracePeriod(node.DeletedAt.Time, tickStart) {
			continue
		}

		if err := w.LockAndProcessNode(node); err != nil {
			w.logger.Errorf("Error processing canvas node %s/%s: %v", node.WorkflowID, node.NodeID, err)
		}
		processed++
	}

	w.logger.WithFields(log.Fields{
		"candidates":  len(nodes),
		"processed":   processed,
		"duration_ms": time.Since(tickStart).Milliseconds(),
	}).Debug("Canvas node cleanup tick completed")
}

func (w *CanvasNodeCleanupWorker) LockAndProcessNode(node models.CanvasNode) error {
	if !node.DeletedAt.Valid || deletedResourceWithinGracePeriod(node.DeletedAt.Time, time.Now()) {
		return nil
	}

	resourcesDeleted := 0
	for resourcesDeleted < w.maxResourcesPerNodePerTick {
		batchLimit := w.deleteBatchSize
		remaining := w.maxResourcesPerNodePerTick - resourcesDeleted
		if batchLimit > remaining {
			batchLimit = remaining
		}

		batchDeleted, done, err := w.processNodeBatch(node, batchLimit)
		if err != nil {
			return err
		}

		resourcesDeleted += batchDeleted
		if done || batchDeleted == 0 {
			return nil
		}

		if w.pauseBetweenBatches > 0 {
			time.Sleep(w.pauseBetweenBatches)
		}
	}

	w.logger.Infof(
		"Partially cleaned node %s from canvas %s (deleted %d resources this pass, more remain)",
		node.NodeID,
		node.WorkflowID,
		resourcesDeleted,
	)
	return nil
}

func (w *CanvasNodeCleanupWorker) processNodeBatch(node models.CanvasNode, batchLimit int) (deleted int, done bool, err error) {
	inFlight := w.inFlightBatches.Add(1)
	for {
		observed := w.maxObservedInFlight.Load()
		if inFlight <= observed || w.maxObservedInFlight.CompareAndSwap(observed, inFlight) {
			break
		}
	}
	defer w.inFlightBatches.Add(-1)

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedNode, lockErr := models.LockDeletedCanvasNode(tx, node.WorkflowID, node.NodeID)
		if lockErr != nil {
			if errors.Is(lockErr, gorm.ErrRecordNotFound) {
				w.logger.Infof("Canvas node %s/%s already being processed - skipping", node.WorkflowID, node.NodeID)
				done = true
				return nil
			}
			return fmt.Errorf("lock deleted canvas node %s/%s: %w", node.WorkflowID, node.NodeID, lockErr)
		}

		if !lockedNode.DeletedAt.Valid || deletedResourceWithinGracePeriod(lockedNode.DeletedAt.Time, time.Now()) {
			done = true
			return nil
		}

		result, cleanErr := models.NewNodeResourceCleaner(tx, lockedNode).
			ForUnreferenced().
			WithLimit(batchLimit).
			Run()
		if cleanErr != nil {
			return fmt.Errorf("failed to delete resources for node %s: %w", lockedNode.NodeID, cleanErr)
		}

		deleted = result.ResourcesDeleted
		if !result.AllDeleted {
			return nil
		}

		if hardDeleteErr := lockedNode.HardDelete(tx); hardDeleteErr != nil {
			return fmt.Errorf("failed to delete canvas node %s: %w", lockedNode.NodeID, hardDeleteErr)
		}

		w.logger.Infof(
			"Deleted node %s from canvas %s (deleted %d resources in final batch)",
			lockedNode.NodeID,
			lockedNode.WorkflowID,
			result.ResourcesDeleted,
		)
		done = true
		return nil
	})
	return deleted, done, err
}
