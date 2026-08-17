package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClosedWorkOrderMetricsRow is one closed work order's contribution to a
// line's metrics. All arithmetic (grouping by line, splitting into current
// vs. prior windows, computing rates/deltas) happens on the Go side so it
// stays unit-testable without a database — see
// pkg/grpc/actions/factories/line_metrics.go.
type ClosedWorkOrderMetricsRow struct {
	WorkOrderID uuid.UUID
	LineID      uuid.UUID
	Result      string
	ClosedAt    time.Time
	CostCents   int64
	ReworkCount int
}

// ListClosedWorkOrderMetricsRows returns one row per closed work order in
// this factory whose `updated_at` (used as a proxy for `closed_at`, see
// docs/design/factory.md) is at or after `since`. Work orders with no
// executions (e.g. abandoned straight from draft) are excluded, since there's
// no line to attribute them to.
//
// Line attribution follows the same convention as the Work Orders table's
// "Line" column: the line of the work order's most recent execution.
//
// `ReworkCount` per work order is the sum of:
//   - `order.comment.added` events authored by a user
//   - `order.status.updated` events sending the order back to `draft`
//   - restarts: dispatches from step 0 beyond the first one
func (f *Factory) ListClosedWorkOrderMetricsRows(tx *gorm.DB, since time.Time) ([]ClosedWorkOrderMetricsRow, error) {
	var rows []ClosedWorkOrderMetricsRow

	err := tx.Raw(`
		WITH last_execution AS (
			SELECT DISTINCT ON (work_order_id) work_order_id, line_id
			FROM factory_work_order_executions
			WHERE factory_id = ?
			ORDER BY work_order_id, created_at DESC, id DESC
		),
		work_order_cost AS (
			SELECT work_order_id, COALESCE(SUM(cost_cents), 0) AS cost_cents
			FROM factory_work_order_executions
			WHERE factory_id = ?
			GROUP BY work_order_id
		),
		restarts AS (
			SELECT work_order_id,
				GREATEST(COUNT(*) FILTER (WHERE step_index = 0) - 1, 0) AS restart_count
			FROM factory_work_order_executions
			WHERE factory_id = ?
			GROUP BY work_order_id
		),
		rework_events AS (
			SELECT e.work_order_id,
				COUNT(*) FILTER (
					WHERE e.type = 'order.comment.added' AND e.data->'author'->>'kind' = 'user'
				) + COUNT(*) FILTER (
					WHERE e.type = 'order.status.updated' AND e.data->>'toState' = 'draft'
				) AS rework_event_count
			FROM factory_work_order_events e
			JOIN factory_work_orders wo ON wo.id = e.work_order_id
			WHERE wo.factory_id = ? AND wo.state = ? AND wo.updated_at >= ?
			GROUP BY e.work_order_id
		)
		SELECT
			wo.id AS work_order_id,
			le.line_id AS line_id,
			wo.result AS result,
			wo.updated_at AS closed_at,
			COALESCE(wc.cost_cents, 0) AS cost_cents,
			COALESCE(re.restart_count, 0) + COALESCE(rw.rework_event_count, 0) AS rework_count
		FROM factory_work_orders wo
		JOIN last_execution le ON le.work_order_id = wo.id
		LEFT JOIN work_order_cost wc ON wc.work_order_id = wo.id
		LEFT JOIN restarts re ON re.work_order_id = wo.id
		LEFT JOIN rework_events rw ON rw.work_order_id = wo.id
		WHERE wo.factory_id = ? AND wo.state = ? AND wo.updated_at >= ?
		ORDER BY wo.updated_at ASC
	`,
		f.ID,
		f.ID,
		f.ID,
		f.ID, FactoryWorkOrderStateClosed, since,
		f.ID, FactoryWorkOrderStateClosed, since,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	return rows, nil
}
