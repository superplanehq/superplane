package telemetry

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
)

//
// Reports metrics at periodic intervals.
//

type Periodic struct {
}

func NewPeriodic() *Periodic {
	return &Periodic{}
}

func (p *Periodic) Start() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.report()
			}
		}
	}()
}

func (p *Periodic) report() {
	p.reportDatabaseLocks()
	p.reportStuckQueueItems()
}

func (p *Periodic) reportDatabaseLocks() {
	if !dbLocksCountHistogramReady.Load() {
		return
	}

	var count int64

	err := database.Conn().Raw("select count(*) from pg_locks").Scan(&count).Error
	if err != nil {
		return
	}

	dbLocksCountHistogram.Record(context.Background(), count)
}

func (p *Periodic) reportStuckQueueItems() {
	if !stuckQueueItemsCountHistogramReady.Load() {
		return
	}

	count, err := countStuckQueueNodes()
	if err != nil {
		return
	}

	queueWorkerStuckItems.Record(context.Background(), count)
}

func countStuckQueueNodes() (int64, error) {
	db := database.Conn()

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
