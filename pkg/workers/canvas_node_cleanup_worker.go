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
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	canvasNodeCleanupTickEvery           = 30 * time.Second
	canvasNodeCleanupMaxNodesPerTick     = 10
	canvasNodeCleanupMaxRunsPerNode      = 10
	canvasNodeCleanupDeleteBatchSize     = 50
	canvasNodeCleanupMaxResourcesPerNode = 200
	canvasNodeCleanupPauseBetweenBatches = 50 * time.Millisecond
)

type CanvasNodeCleanupWorker struct {
	logger                     *log.Entry
	maxNodesPerTick            int
	maxRunsPerNodePerTick      int
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
		maxRunsPerNodePerTick:      canvasNodeCleanupMaxRunsPerNode,
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
	eligibleBefore := tickStart.UTC().AddDate(0, 0, -deletedResourceGracePeriodDays)
	nodes, err := models.ListDeletedCanvasNodes(database.Conn(), eligibleBefore, w.maxNodesPerTick)
	if err != nil {
		w.logger.Errorf("Error finding deleted canvas nodes: %v", err)
		return
	}

	processed := 0
	for _, node := range nodes {
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

	progress := false

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedNode, err := models.LockDeletedCanvasNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.logger.Infof("Canvas node %s/%s already being processed - skipping", node.WorkflowID, node.NodeID)
				progress = true
				return nil
			}
			return fmt.Errorf("lock deleted canvas node %s/%s: %w", node.WorkflowID, node.NodeID, err)
		}

		if !lockedNode.DeletedAt.Valid || deletedResourceWithinGracePeriod(lockedNode.DeletedAt.Time, time.Now()) {
			progress = true
			return nil
		}

		node = *lockedNode
		return nil
	})
	if err != nil {
		return err
	}
	if progress {
		return nil
	}

	runsDeleted, err := w.cleanNodeRuns(node)
	if err != nil {
		return err
	}
	if runsDeleted > 0 {
		progress = true
	}

	remainingRuns, err := node.CountRuns(database.Conn())
	if err != nil {
		return fmt.Errorf("count remaining runs for node %s: %w", node.NodeID, err)
	}
	if remainingRuns > 0 {
		if !progress {
			return w.rotateBlockedNode(node)
		}
		w.logger.Infof(
			"Partially cleaned runs from node %s on canvas %s (deleted %d runs, %d remaining)",
			node.NodeID,
			node.WorkflowID,
			runsDeleted,
			remainingRuns,
		)
		return nil
	}

	resourcesDeleted, complete, err := w.cleanRemainingResources(node)
	if err != nil {
		return err
	}
	if resourcesDeleted > 0 {
		progress = true
	}
	if !complete {
		if !progress {
			return w.rotateBlockedNode(node)
		}
		w.logger.Infof(
			"Partially cleaned remaining resources for node %s on canvas %s (deleted %d resources)",
			node.NodeID,
			node.WorkflowID,
			resourcesDeleted,
		)
		return nil
	}

	hasRemaining, err := node.HasRemainingResources(database.Conn())
	if err != nil {
		return fmt.Errorf("check remaining resources for node %s: %w", node.NodeID, err)
	}
	if hasRemaining {
		return w.rotateBlockedNode(node)
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedNode, err := models.LockDeletedCanvasNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return fmt.Errorf("lock deleted canvas node %s/%s: %w", node.WorkflowID, node.NodeID, err)
		}

		hasRemaining, err := lockedNode.HasRemainingResources(tx)
		if err != nil {
			return fmt.Errorf("check remaining resources for node %s: %w", lockedNode.NodeID, err)
		}
		if hasRemaining {
			return lockedNode.RotateCleanupQueue(tx)
		}

		if err := lockedNode.HardDelete(tx); err != nil {
			return fmt.Errorf("failed to delete canvas node %s: %w", lockedNode.NodeID, err)
		}

		w.logger.Infof("Deleted node %s from canvas %s", lockedNode.NodeID, lockedNode.WorkflowID)
		return nil
	})
}

func (w *CanvasNodeCleanupWorker) cleanNodeRuns(node models.CanvasNode) (int, error) {
	runs, err := node.ListRuns(database.Conn(), w.maxRunsPerNodePerTick)
	if err != nil {
		return 0, fmt.Errorf("list workflow runs for node cleanup: %w", err)
	}

	deleted := 0
	for _, run := range runs {
		logger := logging.WithRun(w.logger, run)

		var summary *models.RunDeletionSummary
		err := w.withTrackedBatch(func() error {
			return database.Conn().Transaction(func(tx *gorm.DB) error {
				locked, err := models.LockCanvasRun(tx, node.WorkflowID, run.ID)
				if err != nil {
					return fmt.Errorf("lock run %s: %w", run.ID, err)
				}
				if locked == nil {
					return nil
				}

				summary, err = locked.DeleteChain(tx)
				if err != nil {
					return fmt.Errorf("delete run chain: %w", err)
				}
				return nil
			})
		})
		if err != nil {
			logger.Errorf("Error cleaning run: %v", err)
			return deleted, err
		}

		if summary != nil {
			logger.WithFields(log.Fields{
				"runs":          summary.Runs,
				"events":        summary.Events,
				"executions":    summary.NodeExecutions,
				"requests":      summary.NodeRequests,
				"execution_kvs": summary.NodeExecutionKVs,
				"queue_items":   summary.NodeQueueItems,
			}).Info("Cleaned run for deleted node")
			deleted++
		}

		if w.pauseBetweenBatches > 0 {
			time.Sleep(w.pauseBetweenBatches)
		}
	}

	return deleted, nil
}

func (w *CanvasNodeCleanupWorker) cleanRemainingResources(node models.CanvasNode) (int, bool, error) {
	totalDeleted := 0
	for totalDeleted < w.maxResourcesPerNodePerTick {
		budget := w.deleteBatchSize
		remaining := w.maxResourcesPerNodePerTick - totalDeleted
		if budget > remaining {
			budget = remaining
		}

		var (
			summary  *models.RunDeletionSummary
			complete bool
		)
		err := w.withTrackedBatch(func() error {
			return database.Conn().Transaction(func(tx *gorm.DB) error {
				var err error
				summary, complete, err = node.DeleteRemainingResources(tx, budget)
				return err
			})
		})
		if err != nil {
			return totalDeleted, false, fmt.Errorf("delete remaining resources for node %s: %w", node.NodeID, err)
		}

		batchDeleted := 0
		if summary != nil {
			batchDeleted = int(summary.TotalRecords())
		}
		totalDeleted += batchDeleted

		if complete || batchDeleted == 0 {
			return totalDeleted, complete || batchDeleted == 0, nil
		}

		if w.pauseBetweenBatches > 0 {
			time.Sleep(w.pauseBetweenBatches)
		}
	}

	return totalDeleted, false, nil
}

func (w *CanvasNodeCleanupWorker) rotateBlockedNode(node models.CanvasNode) error {
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedNode, err := models.LockDeletedCanvasNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		return lockedNode.RotateCleanupQueue(tx)
	})
	if err != nil {
		return fmt.Errorf("rotate blocked canvas node %s: %w", node.NodeID, err)
	}

	w.logger.Infof(
		"No progress cleaning node %s from canvas %s; rotated cleanup queue position",
		node.NodeID,
		node.WorkflowID,
	)
	return nil
}

func (w *CanvasNodeCleanupWorker) withTrackedBatch(fn func() error) error {
	inFlight := w.inFlightBatches.Add(1)
	for {
		observed := w.maxObservedInFlight.Load()
		if inFlight <= observed || w.maxObservedInFlight.CompareAndSwap(observed, inFlight) {
			break
		}
	}
	defer w.inFlightBatches.Add(-1)
	return fn()
}
