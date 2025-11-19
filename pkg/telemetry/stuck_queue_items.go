package telemetry

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
)

func StartStuckQueueItemsReporter(ctx context.Context) {
	if !stuckQueueItemsCountHistogramReady.Load() {
		return
	}

	if !stuckQueueItemsReporterInitializedFlag.CompareAndSwap(false, true) {
		return
	}

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				reportStuckQueueItems(ctx, database.Conn())
			}
		}
	}()
}

func reportStuckQueueItems(ctx context.Context, db *gorm.DB) {
	if !stuckQueueItemsCountHistogramReady.Load() {
		return
	}

	count, err := countStuckQueueNodes(db)
	if err != nil {
		return
	}

	queueWorkerStuckItems.Record(ctx, count)
}

func countStuckQueueNodes(db *gorm.DB) (int64, error) {
	var count int64

	if err := db.
		Raw(`
			SELECT COUNT(*)
			FROM workflow_nodes n
			WHERE EXISTS (
				SELECT 1
				FROM workflow_node_queue_items q
				WHERE q.workflow_id = n.workflow_id
				  AND q.node_id = n.node_id
			)
			AND NOT EXISTS (
				SELECT 1
				FROM workflow_node_executions e
				WHERE e.workflow_id = n.workflow_id
				  AND e.node_id = n.node_id
				  AND e.state <> 'finished'
			)
		`).
		Scan(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
