package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"gorm.io/gorm"
)

// FactoryAutomationRuns is what one automation of a workspace did inside a
// window: how often it ran, how often it failed, how long its finished runs
// took, and what they cost.
type FactoryAutomationRuns struct {
	CanvasID uuid.UUID
	Name     string
	Runs     int
	Failed   int
	// FinishedRuns is the part of Runs that carries a finish time. Duration is
	// averaged over these, so a window full of running work reports no
	// duration instead of a number pulled down by runs that have not ended.
	FinishedRuns    int
	DurationSeconds float64
	CostMicros      int64
}

// AverageDurationHours is the mean wall-clock time of the finished runs.
func (a FactoryAutomationRuns) AverageDurationHours() float64 {
	if a.FinishedRuns == 0 {
		return 0
	}
	return a.DurationSeconds / float64(a.FinishedRuns) / 3600
}

func (a FactoryAutomationRuns) CostCents() int64 {
	return pricebook.MicrosToCents(a.CostMicros)
}

// SummarizeFactoryAutomationRuns reports one row per automation of the
// workspace that ran between `from` and `to`, busiest first.
//
// Runs count against the window they started in, so a row stays comparable
// with the run list of the automation, which is also ordered by start time.
func SummarizeFactoryAutomationRuns(
	tx *gorm.DB,
	factoryID uuid.UUID,
	from, to time.Time,
) ([]FactoryAutomationRuns, error) {
	var rows []FactoryAutomationRuns
	err := tx.Raw(summarizeFactoryAutomationRunsSQL,
		factoryID,
		from, to,
		CanvasRunResultFailed,
		factoryID,
		from, to,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// Spend is recorded per canvas run, so it is summed in its own pass and joined
// back. Summing it in the main query would multiply the cost of a run by the
// number of rows the join produces.
const summarizeFactoryAutomationRunsSQL = `
WITH run_cost AS (
	SELECT r.workflow_id AS workflow_id, COALESCE(SUM(e.cost_micros), 0) AS cost_micros
	FROM workspace_usage_events e
	INNER JOIN workflow_runs r ON r.id = e.canvas_run_id
	INNER JOIN workflows w ON w.id = r.workflow_id
	WHERE w.factory_id = ?
		AND r.created_at >= ? AND r.created_at < ?
	GROUP BY r.workflow_id
)
SELECT
	w.id AS canvas_id,
	w.name AS name,
	COUNT(r.id) AS runs,
	COUNT(r.id) FILTER (WHERE r.result = ?) AS failed,
	COUNT(r.id) FILTER (WHERE r.finished_at IS NOT NULL) AS finished_runs,
	COALESCE(
		SUM(EXTRACT(EPOCH FROM (r.finished_at - r.created_at))) FILTER (WHERE r.finished_at IS NOT NULL),
		0
	) AS duration_seconds,
	COALESCE(c.cost_micros, 0) AS cost_micros
FROM workflows w
INNER JOIN workflow_runs r ON r.workflow_id = w.id
LEFT JOIN run_cost c ON c.workflow_id = w.id
WHERE w.factory_id = ?
	AND w.deleted_at IS NULL
	AND r.created_at >= ? AND r.created_at < ?
GROUP BY w.id, w.name, c.cost_micros
ORDER BY runs DESC, w.name ASC
`
