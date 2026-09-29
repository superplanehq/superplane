package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

// ClosedWorkOrderMetricRow is one closed work order attributed to a line,
// with the timestamps and cost the line-metrics aggregator needs.
type ClosedWorkOrderMetricRow struct {
	WorkOrderID      uuid.UUID
	LineID           uuid.UUID
	ClosedAt         time.Time
	Merged           bool
	MergedAt         *time.Time
	ExecutionMinutes float64
	CostCents        int64
}

// ListClosedWorkOrderMetricRows returns one row per closed work order in
// [from, to) for the factory. Line attribution is the latest execution.
// Work orders with no executions are excluded. Close time is the latest
// order.status.updated event with toState=closed, not updated_at.
//
// A row is merged when the work order result is completed, or when a
// factory pull request is stamped merged (merged_at or state=merged).
// Merge time is the pull request merged_at, or the close event when that
// stamp is missing.
func ListClosedWorkOrderMetricRows(tx *gorm.DB, factoryID uuid.UUID, from, to time.Time) ([]ClosedWorkOrderMetricRow, error) {
	var rows []ClosedWorkOrderMetricRow
	err := tx.Raw(listClosedWorkOrderMetricRowsSQL,
		factoryID,
		factory.EventTypeOrderStatusUpdated,
		FactoryWorkOrderStateClosed,
		factoryID,
		factoryID,
		FactoryPullRequestStateMerged,
		factoryID,
		FactoryWorkOrderResultCompleted,
		FactoryWorkOrderResultCompleted,
		factoryID,
		FactoryWorkOrderStateClosed,
		from,
		to,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

const listClosedWorkOrderMetricRowsSQL = `
WITH latest_close AS (
	SELECT DISTINCT ON (e.work_order_id)
		e.work_order_id,
		e.created_at AS closed_at
	FROM factory_work_order_events e
	INNER JOIN factory_work_orders wo ON wo.id = e.work_order_id
	WHERE wo.factory_id = ?
		AND e.type = ?
		AND e.data->>'toState' = ?
	ORDER BY e.work_order_id, e.created_at DESC
),
latest_execution AS (
	SELECT DISTINCT ON (x.work_order_id)
		x.work_order_id,
		x.line_id
	FROM factory_work_order_executions x
	WHERE x.factory_id = ?
	ORDER BY x.work_order_id, x.created_at DESC, x.id DESC
),
execution_stats AS (
	SELECT
		x.work_order_id,
		COALESCE(SUM(
			CASE
				WHEN x.finished_at IS NOT NULL AND x.finished_at > x.created_at
				THEN EXTRACT(EPOCH FROM (x.finished_at - x.created_at)) / 60.0
				ELSE 0
			END
		), 0) AS execution_minutes,
		COALESCE(SUM(x.cost_cents), 0) AS cost_cents
	FROM factory_work_order_executions x
	WHERE x.factory_id = ?
	GROUP BY x.work_order_id
),
merged_pr AS (
	SELECT
		p.work_order_id,
		BOOL_OR(
			p.merged_at IS NOT NULL
			OR p.state = ?
		) AS merged,
		MIN(p.merged_at) AS merged_at
	FROM factory_pull_requests p
	WHERE p.factory_id = ?
	GROUP BY p.work_order_id
)
SELECT
	le.line_id,
	wo.id AS work_order_id,
	lc.closed_at,
	(COALESCE(mp.merged, FALSE) OR wo.result = ?) AS merged,
	COALESCE(
		mp.merged_at,
		CASE WHEN COALESCE(mp.merged, FALSE) OR wo.result = ? THEN lc.closed_at END
	) AS merged_at,
	es.execution_minutes,
	es.cost_cents
FROM factory_work_orders wo
INNER JOIN latest_close lc ON lc.work_order_id = wo.id
INNER JOIN latest_execution le ON le.work_order_id = wo.id
INNER JOIN execution_stats es ON es.work_order_id = wo.id
LEFT JOIN merged_pr mp ON mp.work_order_id = wo.id
WHERE wo.factory_id = ?
	AND wo.state = ?
	AND lc.closed_at >= ?
	AND lc.closed_at < ?
`
