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
	ctx context.Context
}

func NewPeriodic(ctx context.Context) *Periodic {
	return &Periodic{
		ctx: ctx,
	}
}

func (p *Periodic) Start() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			p.report()
		}
	}()
}

func (p *Periodic) report() {
	p.reportDatabaseLocks()
	p.reportLongQueries()
	p.reportStuckQueueItems()
}

func (p *Periodic) reportDatabaseLocks() {
	var count int64

	err := database.Conn().Raw("select count(*) from pg_locks").Scan(&count).Error
	if err != nil {
		return
	}

	RecordDBLocksCount(p.ctx, count)
}

func (p *Periodic) reportStuckQueueItems() {
	count, err := countStuckQueueNodes()
	if err != nil {
		return
	}

	RecordStuckQueueItemsCount(p.ctx, int(count))
}

func (p *Periodic) reportLongQueries() {
	var count int64

	err := database.Conn().Raw(`
		SELECT COUNT(*)
		FROM pg_stat_activity
		WHERE state = 'active'
		  AND now() - query_start > interval '1 minutes'
	`).Scan(&count).Error

	if err != nil {
		return
	}

	RecordDBLongQueriesCount(p.ctx, count)
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
