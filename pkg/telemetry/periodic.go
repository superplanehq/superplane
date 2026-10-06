package telemetry

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

//
// Reports metrics at periodic intervals.
//

const periodicMetricsInterval = 60 * time.Second

type Periodic struct {
	ctx                  context.Context
	lastPoolWaitCount    int64
	lastPoolWaitDuration time.Duration
}

func NewPeriodic(ctx context.Context) *Periodic {
	return &Periodic{
		ctx: ctx,
	}
}

func (p *Periodic) Start() {
	go p.reportEvery(periodicMetricsInterval, p.report)
}

func (p *Periodic) reportEvery(interval time.Duration, report func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			report()
		}
	}
}

func (p *Periodic) report() {
	p.reportDatabasePoolStats()
	p.reportDatabaseLocks()
	p.reportLongQueries()
	p.reportStuckQueueItems()
	p.reportPendingEvents()
	p.reportPendingExecutions()
	p.reportRunnerCounts()
	p.reportRunnerTaskCounts()
}

func (p *Periodic) reportDatabasePoolStats() {
	stats, err := database.PoolStats()
	if err != nil {
		return
	}

	RecordDBPoolStats(
		p.ctx,
		int64(stats.MaxOpenConnections),
		int64(stats.OpenConnections),
		int64(stats.InUse),
		int64(stats.Idle),
	)

	waitCountDelta := stats.WaitCount - p.lastPoolWaitCount
	waitDurationDelta := stats.WaitDuration - p.lastPoolWaitDuration
	p.lastPoolWaitCount = stats.WaitCount
	p.lastPoolWaitDuration = stats.WaitDuration

	RecordDBPoolWaitCount(p.ctx, waitCountDelta)
	RecordDBPoolWaitDuration(p.ctx, waitDurationDelta)
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

func (p *Periodic) reportPendingEvents() {
	count, err := countPendingEvents()
	if err != nil {
		return
	}

	RecordPendingEventsCount(p.ctx, count)
}

func (p *Periodic) reportPendingExecutions() {
	count, err := countPendingExecutions()
	if err != nil {
		return
	}

	RecordPendingExecutionsCount(p.ctx, count)
}

func (p *Periodic) reportRunnerCounts() {
	counts, err := models.ListRunnerCountsByFleetState(database.DB(p.ctx))
	if err != nil {
		return
	}

	for _, count := range counts {
		recordRunnerCount(p.ctx, count)
	}
}

func (p *Periodic) reportRunnerTaskCounts() {
	counts, err := models.ListRunnerTaskCountsByFleetState(database.DB(p.ctx))
	if err != nil {
		return
	}

	for _, count := range counts {
		recordRunnerTaskCount(p.ctx, count)
	}
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

func countPendingEvents() (int64, error) {
	var count int64

	err := database.Conn().
		Table("workflow_events AS we").
		Joins("JOIN workflows AS w ON we.workflow_id = w.id").
		Joins("JOIN organizations AS o ON w.organization_id = o.id").
		Where("we.state = ?", "pending").
		Where("w.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countPendingExecutions() (int64, error) {
	var count int64

	err := database.Conn().
		Table("workflow_node_executions AS wne").
		Joins("JOIN workflows AS w ON wne.workflow_id = w.id").
		Joins("JOIN organizations AS o ON w.organization_id = o.id").
		Where("wne.state = ?", "pending").
		Where("w.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}
