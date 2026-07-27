package workers

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	nodeRequestCleanupTickEvery           = 1 * time.Minute
	nodeRequestCleanupRetentionDays       = 7
	nodeRequestCleanupDeleteBatchSize     = 500
	nodeRequestCleanupMaxDeletesPerTick   = 5000
	nodeRequestCleanupPauseBetweenBatches = 50 * time.Millisecond
)

// NodeRequestCleanupWorker deletes completed workflow_node_requests after a
// retention window. Processing only marks requests completed, so this table
// otherwise grows without bound.
type NodeRequestCleanupWorker struct {
	logger              *log.Entry
	retentionDays       int
	deleteBatchSize     int
	maxDeletesPerTick   int
	pauseBetweenBatches time.Duration
}

func NewNodeRequestCleanupWorker() *NodeRequestCleanupWorker {
	return &NodeRequestCleanupWorker{
		logger:              log.WithFields(log.Fields{"worker": "NodeRequestCleanupWorker"}),
		retentionDays:       nodeRequestCleanupRetentionDays,
		deleteBatchSize:     nodeRequestCleanupDeleteBatchSize,
		maxDeletesPerTick:   nodeRequestCleanupMaxDeletesPerTick,
		pauseBetweenBatches: nodeRequestCleanupPauseBetweenBatches,
	}
}

func (w *NodeRequestCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(nodeRequestCleanupTickEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *NodeRequestCleanupWorker) tick(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	startedAt := time.Now()
	olderThan := startedAt.AddDate(0, 0, -w.retentionDays)
	deleted, err := w.cleanCompletedRequests(olderThan, w.maxDeletesPerTick)
	if err != nil {
		w.logger.Errorf("Error cleaning completed node requests: %v", err)
		return
	}

	if deleted == 0 {
		return
	}

	logger := w.logger.WithFields(log.Fields{
		"deleted":     deleted,
		"older_than":  olderThan.UTC().Format(time.RFC3339),
		"duration_ms": time.Since(startedAt).Milliseconds(),
	})

	if deleted >= int64(w.maxDeletesPerTick) {
		logger.Warn("Node request cleanup reached the per-tick limit; more completed requests may remain")
		return
	}

	logger.Info("Deleted completed node requests")
}

func (w *NodeRequestCleanupWorker) cleanCompletedRequests(olderThan time.Time, limit int) (int64, error) {
	totalDeleted := int64(0)

	for totalDeleted < int64(limit) {
		budget := w.deleteBatchSize
		remaining := limit - int(totalDeleted)
		if budget > remaining {
			budget = remaining
		}

		var deleted int64
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			count, err := models.DeleteExpiredCompletedNodeRequests(tx, olderThan, budget)
			if err != nil {
				return err
			}
			deleted = count
			return nil
		})
		if err != nil {
			return totalDeleted, fmt.Errorf("delete expired completed node requests: %w", err)
		}

		totalDeleted += deleted
		if deleted == 0 {
			return totalDeleted, nil
		}

		if w.pauseBetweenBatches > 0 {
			time.Sleep(w.pauseBetweenBatches)
		}
	}

	return totalDeleted, nil
}
