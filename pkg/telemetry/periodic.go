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
	p.reportRunnerFleetCounts()
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

func (p *Periodic) reportRunnerFleetCounts() {
	counts, err := listRunnerFleetCounts()
	if err != nil {
		return
	}

	for _, count := range counts {
		organizationID := ""
		if count.ScopeType == models.RunnerFleetScopeOrganization && count.ScopeID != nil {
			organizationID = *count.ScopeID
		}
		RecordRunnerFleetCount(p.ctx, count.FleetSlug, organizationID, models.RunnerStateIdle, count.IdleCount)
		RecordRunnerFleetCount(p.ctx, count.FleetSlug, organizationID, models.RunnerStateBusy, count.BusyCount)
	}
}

type runnerFleetCount struct {
	FleetSlug string
	ScopeType string
	ScopeID   *string
	IdleCount int64
	BusyCount int64
}

func listRunnerFleetCounts() ([]runnerFleetCount, error) {
	var rows []runnerFleetCount
	err := database.Conn().Raw(`
		SELECT
			fleets.slug AS fleet_slug,
			fleets.scope_type AS scope_type,
			CAST(fleets.scope_id AS text) AS scope_id,
			COUNT(runners.id) FILTER (WHERE runners.state = ?) AS idle_count,
			COUNT(runners.id) FILTER (WHERE runners.state = ?) AS busy_count
		FROM runner_fleets AS fleets
		LEFT JOIN runners
			ON runners.fleet_id = fleets.id
			AND runners.state IN (?, ?)
		WHERE fleets.deleted_at IS NULL
		GROUP BY fleets.id, fleets.slug, fleets.scope_type, fleets.scope_id
	`,
		models.RunnerStateIdle,
		models.RunnerStateBusy,
		models.RunnerStateIdle,
		models.RunnerStateBusy,
	).Scan(&rows).Error
	return rows, err
}

func (p *Periodic) reportPendingExecutions() {
	count, err := countPendingExecutions()
	if err != nil {
		return
	}

	RecordPendingExecutionsCount(p.ctx, count)
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
