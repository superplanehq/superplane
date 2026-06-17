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
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			p.report()
		}
	}()
}

func (p *Periodic) report() {
	p.reportDatabasePoolStats()
	p.reportDatabaseLocks()
	p.reportLongQueries()
	p.reportStuckQueueItems()
	p.reportPendingEvents()
	p.reportPendingExecutions()
	p.reportPendingIntegrationRequests()
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

func (p *Periodic) reportPendingIntegrationRequests() {
	total, maxPerInstallation, err := countPendingIntegrationRequests()
	if err != nil {
		return
	}

	RecordPendingIntegrationRequestsCount(p.ctx, total)
	RecordPendingIntegrationRequestsMaxPerInstallation(p.ctx, maxPerInstallation)
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
		Where("we.state = ?", "pending").
		Where("w.deleted_at IS NULL").
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
		Where("wne.state = ?", "pending").
		Where("w.deleted_at IS NULL").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// countPendingIntegrationRequests returns the total number of pending integration
// requests and the largest number held by any single installation. The
// max-per-installation value is the real early warning for a runaway scheduling
// loop like #5386, where one installation accumulates many self-rescheduling
// chains while the global total still looks unremarkable.
func countPendingIntegrationRequests() (int64, int64, error) {
	type result struct {
		Total              int64
		MaxPerInstallation int64
	}

	var stats result
	err := database.Conn().
		Raw(`
			SELECT
				COALESCE(SUM(per_installation.count), 0) AS total,
				COALESCE(MAX(per_installation.count), 0) AS max_per_installation
			FROM (
				SELECT r.app_installation_id, COUNT(*) AS count
				FROM app_installation_requests AS r
				JOIN app_installations AS i ON r.app_installation_id = i.id
				WHERE r.state = 'pending'
				  AND i.deleted_at IS NULL
				GROUP BY r.app_installation_id
			) AS per_installation
		`).
		Scan(&stats).
		Error
	if err != nil {
		return 0, 0, err
	}

	return stats.Total, stats.MaxPerInstallation, nil
}
