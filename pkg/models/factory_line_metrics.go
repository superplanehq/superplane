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
	FirstExecutionAt *time.Time
	CostCents        int64
}

// ListClosedWorkOrderMetricRows returns one row per closed work order in
// [from, to) for the factory. Line attribution is the latest execution.
// Work orders with no executions are excluded. Close time is the latest
// order.status.updated event with toState=closed, not updated_at.
func ListClosedWorkOrderMetricRows(tx *gorm.DB, factoryID uuid.UUID, from, to time.Time) ([]ClosedWorkOrderMetricRow, error) {
	var rows []ClosedWorkOrderMetricRow
	err := tx.Raw(listClosedWorkOrderMetricRowsSQL,
		factoryID,
		factory.EventTypeOrderStatusUpdated,
		FactoryWorkOrderStateClosed,
		factoryID,
		factoryID,
		PrArtifactStateMerged,
		factoryID,
		FactoryWorkOrderArtifactTypePR,
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
		MIN(x.created_at) AS first_execution_at,
		COALESCE(SUM(x.cost_cents), 0) AS cost_cents
	FROM factory_work_order_executions x
	WHERE x.factory_id = ?
	GROUP BY x.work_order_id
),
merged_pr AS (
	SELECT
		a.work_order_id,
		BOOL_OR(a.merged_at IS NOT NULL OR COALESCE(a.data->>'state', '') = ?) AS merged,
		MIN(a.merged_at) AS merged_at
	FROM factory_work_order_artifacts a
	WHERE a.factory_id = ?
		AND a.type = ?
	GROUP BY a.work_order_id
)
SELECT
	le.line_id,
	wo.id AS work_order_id,
	lc.closed_at,
	COALESCE(mp.merged, FALSE) AS merged,
	mp.merged_at,
	es.first_execution_at,
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
