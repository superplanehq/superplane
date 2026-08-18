package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

const (
	FactoryLineMetricsWindowDays = 30
	FactoryLineMetricsTrendDays  = 14
)

// FactoryLineMetrics is the last-30-day performance rollup for one line.
// Present is false when the line has no closed work orders in the window.
type FactoryLineMetrics struct {
	LineID             uuid.UUID
	Present            bool
	SuccessRatePct     float64
	MergedCount        int
	TotalClosedCount   int
	ReworkPerWorkOrder float64
	CostPerSuccessUsd  float64
	SuccessTrendPct    []float64
	SuccessDeltaPts    float64
	ReworkDelta        float64
	CostDeltaUsd       float64
	ThroughputPerDay   float64
	ThroughputTrend    []int
}

type closedWorkOrderRow struct {
	ID       uuid.UUID
	LineID   uuid.UUID
	ClosedAt time.Time
}

type workOrderIntRow struct {
	WorkOrderID uuid.UUID
	Value       int64
}

type workOrderCostRow struct {
	WorkOrderID uuid.UUID
	CostCents   int64
	ExtraRuns   int64
}

type windowStats struct {
	closed      int
	merged      int
	successRate float64
	rework      float64
	costPerUSD  float64
}

// ListFactoryLineMetrics returns one row per factory line. now is the
// exclusive end of the current 30-day window.
func ListFactoryLineMetrics(tx *gorm.DB, orgID, factoryID uuid.UUID, now time.Time) ([]FactoryLineMetrics, error) {
	lines, err := listFactoryLineIDs(tx, orgID, factoryID)
	if err != nil {
		return nil, err
	}

	currentStart := now.AddDate(0, 0, -FactoryLineMetricsWindowDays)
	priorStart := currentStart.AddDate(0, 0, -FactoryLineMetricsWindowDays)

	closed, err := listClosedWorkOrdersOnLines(tx, orgID, factoryID, priorStart, now)
	if err != nil {
		return nil, err
	}

	orderIDs := make([]uuid.UUID, 0, len(closed))
	for _, row := range closed {
		orderIDs = append(orderIDs, row.ID)
	}

	merged, err := listMergedWorkOrderIDs(tx, orgID, factoryID, orderIDs)
	if err != nil {
		return nil, err
	}
	comments, err := countUserCommentsByWorkOrder(tx, orderIDs)
	if err != nil {
		return nil, err
	}
	costs, err := listWorkOrderReworkAndCost(tx, orderIDs)
	if err != nil {
		return nil, err
	}

	byLine := map[uuid.UUID][]closedWorkOrderRow{}
	for _, row := range closed {
		byLine[row.LineID] = append(byLine[row.LineID], row)
	}

	result := make([]FactoryLineMetrics, 0, len(lines))
	for _, lineID := range lines {
		result = append(result, buildLineMetrics(
			lineID,
			byLine[lineID],
			merged,
			comments,
			costs,
			currentStart,
			now,
		))
	}
	return result, nil
}

func listFactoryLineIDs(tx *gorm.DB, orgID, factoryID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := tx.
		Model(&FactoryLine{}).
		Where("organization_id = ? AND factory_id = ?", orgID, factoryID).
		Order("name ASC").
		Order("id ASC").
		Pluck("id", &ids).
		Error
	return ids, err
}

func listClosedWorkOrdersOnLines(
	tx *gorm.DB,
	orgID, factoryID uuid.UUID,
	from, until time.Time,
) ([]closedWorkOrderRow, error) {
	var rows []closedWorkOrderRow
	err := tx.Raw(`
		WITH latest_exec AS (
			SELECT DISTINCT ON (e.work_order_id)
				e.work_order_id,
				e.line_id
			FROM factory_work_order_executions e
			WHERE e.organization_id = ? AND e.factory_id = ?
			ORDER BY e.work_order_id, e.created_at DESC, e.id DESC
		),
		latest_close AS (
			SELECT DISTINCT ON (ev.work_order_id)
				ev.work_order_id,
				ev.created_at AS closed_at
			FROM factory_work_order_events ev
			INNER JOIN factory_work_orders o ON o.id = ev.work_order_id
			WHERE o.organization_id = ?
				AND o.factory_id = ?
				AND ev.type = ?
				AND ev.data->>'toState' = ?
			ORDER BY ev.work_order_id, ev.created_at DESC, ev.id DESC
		)
		SELECT o.id, le.line_id, lc.closed_at
		FROM factory_work_orders o
		INNER JOIN latest_exec le ON le.work_order_id = o.id
		INNER JOIN latest_close lc ON lc.work_order_id = o.id
		WHERE o.organization_id = ?
			AND o.factory_id = ?
			AND o.state = ?
			AND lc.closed_at >= ?
			AND lc.closed_at < ?
	`, orgID, factoryID, orgID, factoryID, factory.EventTypeOrderStatusUpdated, FactoryWorkOrderStateClosed,
		orgID, factoryID, FactoryWorkOrderStateClosed, from, until).
		Scan(&rows).Error
	return rows, err
}

func listMergedWorkOrderIDs(tx *gorm.DB, orgID, factoryID uuid.UUID, orderIDs []uuid.UUID) (map[uuid.UUID]struct{}, error) {
	result := map[uuid.UUID]struct{}{}
	if len(orderIDs) == 0 {
		return result, nil
	}

	var ids []uuid.UUID
	err := tx.
		Model(&FactoryWorkOrderArtifact{}).
		Where("organization_id = ? AND factory_id = ? AND type = ? AND data->>'state' = ? AND work_order_id IN ?",
			orgID, factoryID, FactoryWorkOrderArtifactTypePR, PrArtifactStateMerged, orderIDs).
		Distinct("work_order_id").
		Pluck("work_order_id", &ids).
		Error
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		result[id] = struct{}{}
	}
	return result, nil
}

func countUserCommentsByWorkOrder(tx *gorm.DB, orderIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	result := map[uuid.UUID]int64{}
	if len(orderIDs) == 0 {
		return result, nil
	}

	var rows []workOrderIntRow
	err := tx.Raw(`
		SELECT ev.work_order_id, COUNT(*) AS value
		FROM factory_work_order_events ev
		WHERE ev.type = ?
			AND ev.data->'author'->>'kind' = ?
			AND ev.work_order_id IN ?
		GROUP BY ev.work_order_id
	`, factory.EventTypeOrderCommentAdded, factory.CommentAuthorKindUser, orderIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.WorkOrderID] = row.Value
	}
	return result, nil
}

func listWorkOrderReworkAndCost(tx *gorm.DB, orderIDs []uuid.UUID) (map[uuid.UUID]workOrderCostRow, error) {
	result := map[uuid.UUID]workOrderCostRow{}
	if len(orderIDs) == 0 {
		return result, nil
	}

	var rows []workOrderCostRow
	err := tx.Raw(`
		SELECT
			work_order_id,
			SUM(cost_cents) AS cost_cents,
			SUM(GREATEST(cnt - 1, 0)) AS extra_runs
		FROM (
			SELECT work_order_id, step_index, COUNT(*) AS cnt, SUM(cost_cents) AS cost_cents
			FROM factory_work_order_executions
			WHERE work_order_id IN ?
			GROUP BY work_order_id, step_index
		) grouped
		GROUP BY work_order_id
	`, orderIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.WorkOrderID] = row
	}
	return result, nil
}

func buildLineMetrics(
	lineID uuid.UUID,
	closed []closedWorkOrderRow,
	merged map[uuid.UUID]struct{},
	comments map[uuid.UUID]int64,
	costs map[uuid.UUID]workOrderCostRow,
	currentStart, now time.Time,
) FactoryLineMetrics {
	metrics := FactoryLineMetrics{
		LineID:          lineID,
		SuccessTrendPct: make([]float64, FactoryLineMetricsTrendDays),
		ThroughputTrend: make([]int, FactoryLineMetricsTrendDays),
	}

	var current, prior []closedWorkOrderRow
	for _, row := range closed {
		if !row.ClosedAt.Before(currentStart) && row.ClosedAt.Before(now) {
			current = append(current, row)
			continue
		}
		if row.ClosedAt.Before(currentStart) {
			prior = append(prior, row)
		}
	}

	if len(current) == 0 {
		return metrics
	}

	currentStats := statsForWindow(current, merged, comments, costs)
	priorStats := statsForWindow(prior, merged, comments, costs)
	fillTrend(metrics.SuccessTrendPct, metrics.ThroughputTrend, current, merged, now)

	metrics.Present = true
	metrics.SuccessRatePct = currentStats.successRate
	metrics.MergedCount = currentStats.merged
	metrics.TotalClosedCount = currentStats.closed
	metrics.ReworkPerWorkOrder = currentStats.rework
	metrics.CostPerSuccessUsd = currentStats.costPerUSD
	metrics.SuccessDeltaPts = currentStats.successRate - priorStats.successRate
	metrics.ReworkDelta = currentStats.rework - priorStats.rework
	metrics.CostDeltaUsd = currentStats.costPerUSD - priorStats.costPerUSD
	metrics.ThroughputPerDay = float64(currentStats.merged) / float64(FactoryLineMetricsWindowDays)
	return metrics
}

func statsForWindow(
	closed []closedWorkOrderRow,
	merged map[uuid.UUID]struct{},
	comments map[uuid.UUID]int64,
	costs map[uuid.UUID]workOrderCostRow,
) windowStats {
	stats := windowStats{closed: len(closed)}
	if stats.closed == 0 {
		return stats
	}

	var interventions int64
	var costCents int64
	for _, row := range closed {
		if _, ok := merged[row.ID]; ok {
			stats.merged++
		}
		interventions += comments[row.ID]
		cost := costs[row.ID]
		interventions += cost.ExtraRuns
		costCents += cost.CostCents
	}

	stats.successRate = (float64(stats.merged) / float64(stats.closed)) * 100
	stats.rework = float64(interventions) / float64(stats.closed)
	if stats.merged > 0 {
		stats.costPerUSD = float64(costCents) / float64(stats.merged) / 100
	}
	return stats
}

func fillTrend(
	success []float64,
	throughput []int,
	closed []closedWorkOrderRow,
	merged map[uuid.UUID]struct{},
	now time.Time,
) {
	type dayCounts struct {
		closed int
		merged int
	}
	byDay := map[time.Time]dayCounts{}
	for _, row := range closed {
		day := utcDay(row.ClosedAt)
		counts := byDay[day]
		counts.closed++
		if _, ok := merged[row.ID]; ok {
			counts.merged++
		}
		byDay[day] = counts
	}

	end := utcDay(now.Add(-time.Nanosecond))
	start := end.AddDate(0, 0, -(FactoryLineMetricsTrendDays - 1))
	for i := 0; i < FactoryLineMetricsTrendDays; i++ {
		day := start.AddDate(0, 0, i)
		counts := byDay[day]
		throughput[i] = counts.merged
		if counts.closed > 0 {
			success[i] = (float64(counts.merged) / float64(counts.closed)) * 100
		}
	}
}

func utcDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}
