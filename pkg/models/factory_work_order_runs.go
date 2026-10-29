package models

import (
	"bytes"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// DefaultTaskRunHistoryLimit is the run-history page size when the caller omits one.
	DefaultTaskRunHistoryLimit = 50
	// MaxTaskRunHistoryLimit is the largest run-history page.
	MaxTaskRunHistoryLimit = 200
)

// taskRunIDsCTE is the set of runs tied to one task.
// It includes planning, line steps, pull-request activity, and descendants.
// A creator run shared by another task is left out, including as a root for descendants.
// factoryTaskRunAssociationsCTE is shared by routing, cancellation and owner
// restrictions. Only stored task links carry authority; payloads do not.
const factoryTaskRunAssociationsCTE = `
WITH RECURSIVE scoped_runs AS (
 SELECT r.id, r.parent_run_id FROM workflow_runs r
 JOIN workflows w ON w.id = r.workflow_id
 WHERE w.organization_id = ? AND w.factory_id = ?
), direct_task_runs AS (
 SELECT e.work_order_id, r.id FROM scoped_runs r
 JOIN factory_work_order_executions e ON e.run_id = r.id
 UNION
 SELECT s.draft_work_order_id, r.id FROM scoped_runs r
 JOIN factory_planning_sessions s ON s.canvas_run_id = r.id
 WHERE s.draft_work_order_id IS NOT NULL
 UNION
 SELECT f.work_order_id, r.id FROM scoped_runs r
 JOIN factory_work_order_refinement_runs f ON f.run_id = r.id
 UNION
 SELECT p.work_order_id, r.id FROM scoped_runs r
 JOIN factory_pull_request_runs a ON a.run_id = r.id
 JOIN factory_pull_requests p ON p.id = a.pull_request_id
 UNION
 SELECT o.id, r.id FROM scoped_runs r
 JOIN factory_work_orders o ON o.source_run_id = r.id
 WHERE NOT EXISTS (
  SELECT 1 FROM factory_work_orders shared
  WHERE shared.source_run_id = o.source_run_id AND shared.factory_id = o.factory_id AND shared.id <> o.id
 )
), associated_task_runs AS (
 SELECT work_order_id, id FROM direct_task_runs
 UNION
 SELECT a.work_order_id, r.id FROM scoped_runs r
 JOIN associated_task_runs a ON r.parent_run_id = a.id
)`

const taskRunIDsCTE = factoryTaskRunAssociationsCTE + `,
task_runs AS (SELECT id FROM associated_task_runs WHERE work_order_id = ?)`

const legacyTaskRunIDsCTE = `
WITH RECURSIVE shared_creator AS (
	SELECT source_run_id AS id
	FROM factory_work_orders
	WHERE factory_id = ? AND source_run_id = ? AND id <> ? AND source_run_id IS NOT NULL
),
task_runs AS (
	SELECT r.id FROM workflow_runs r
	JOIN workflows w ON w.id = r.workflow_id
	WHERE w.organization_id = ? AND w.factory_id = ?
	  AND r.id NOT IN (SELECT id FROM shared_creator)
	  AND (
		r.id = ?
		OR r.id IN (SELECT run_id FROM factory_work_order_executions WHERE work_order_id = ?)
		OR r.id IN (SELECT canvas_run_id FROM factory_planning_sessions WHERE draft_work_order_id = ?)
		OR r.id IN (
			SELECT a.run_id FROM factory_pull_request_runs a
			JOIN factory_pull_requests p ON p.id = a.pull_request_id WHERE p.work_order_id = ?
		)
		OR r.input->'work_order'->>'id' = ?
		OR r.id IN (
			SELECT run_id FROM workflow_events
			WHERE workflow_id IN (SELECT id FROM workflows WHERE factory_id = ?) AND (
				COALESCE(data->'data', data)->'workOrder'->>'id' = ?
				OR COALESCE(data->'data', data)->'work_order'->>'id' = ?
			)
		)
	)
	UNION
	SELECT r.id FROM workflow_runs r
	JOIN task_runs t ON r.parent_run_id = t.id
	WHERE r.id NOT IN (SELECT id FROM shared_creator)
)`

// TaskRunCursor is the keyset position of a run-history page.
// The next page is older than this run in created_at DESC, id DESC order.
type TaskRunCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// TaskRunHistoryQuery selects one page of associated runs.
type TaskRunHistoryQuery struct {
	Limit  int
	Before *TaskRunCursor
}

// TaskRun is one associated automation run.
type TaskRun struct {
	ID                  uuid.UUID
	AutomationID        uuid.UUID
	AutomationName      string
	ParentRunID         *uuid.UUID
	State               string
	Result              string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	FinishedAt          *time.Time
	ActivityTitle       string
	ActivityDescription string
	UsageRecorded       bool
	Usage               UsageTotals
}

// TaskRunCount is the number of unique runs for one automation and result.
type TaskRunCount struct {
	AutomationID   uuid.UUID
	AutomationName string
	Result         string
	Count          int64
}

// TaskRunHistory is one page of runs plus counts for the full association set.
type TaskRunHistory struct {
	Runs        []TaskRun
	Counts      []TaskRunCount
	HasNextPage bool
}

// ListRuns includes task automations, line steps, PR activity, and their descendants.
// A creator run shared by other tasks is left out, including as a root for descendants.
func (o *FactoryWorkOrder) ListRuns(tx *gorm.DB) ([]CanvasRun, error) {
	query, args, err := o.taskRunQuery(tx)
	if err != nil {
		return nil, err
	}
	var runs []CanvasRun
	err = tx.Raw(
		query+" SELECT r.* FROM workflow_runs r JOIN task_runs t ON r.id = t.id ORDER BY r.id",
		args...,
	).Scan(&runs).Error
	return runs, err
}

// ListTaskRunHistory returns one page of associated runs and full-history counts.
// Counts are grouped by automation canvas and run result. They cover every associated run.
func (o *FactoryWorkOrder) ListTaskRunHistory(tx *gorm.DB, query TaskRunHistoryQuery) (TaskRunHistory, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = DefaultTaskRunHistoryLimit
	}
	if limit > MaxTaskRunHistoryLimit {
		limit = MaxTaskRunHistoryLimit
	}

	querySQL, args, err := o.taskRunQuery(tx)
	if err != nil {
		return TaskRunHistory{}, err
	}
	cursorSQL := ""
	if query.Before != nil {
		cursorSQL = "WHERE (r.created_at, r.id) < (?, ?)"
		args = append(args, query.Before.CreatedAt, query.Before.ID)
	}
	args = append(args, limit+1)

	var rows []taskRunRow
	err = tx.Raw(querySQL+`
		SELECT r.id, r.workflow_id, w.name AS workflow_name, r.parent_run_id,
			r.state, r.result, r.created_at, r.updated_at, r.finished_at,
			COALESCE(activity.title, '') AS activity_title,
			COALESCE(activity.description, '') AS activity_description
		FROM workflow_runs r
		JOIN task_runs t ON r.id = t.id
		JOIN workflows w ON w.id = r.workflow_id
		LEFT JOIN LATERAL (
			SELECT a.title, a.description
			FROM factory_pull_request_runs a
			WHERE a.run_id = r.id
			LIMIT 1
		) activity ON true
		`+cursorSQL+`
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT ?
	`, args...).Scan(&rows).Error
	if err != nil {
		return TaskRunHistory{}, err
	}

	hasNext := len(rows) > limit
	if hasNext {
		rows = rows[:limit]
	}

	counts, err := o.listTaskRunCounts(tx)
	if err != nil {
		return TaskRunHistory{}, err
	}

	usageByRun := map[uuid.UUID]UsageTotals{}
	if len(rows) > 0 {
		ids := make([]uuid.UUID, len(rows))
		for i := range rows {
			ids[i] = rows[i].ID
		}
		usageByRun, err = SumUsageForCanvasRuns(tx, ids)
		if err != nil {
			return TaskRunHistory{}, err
		}
	}

	runs := make([]TaskRun, len(rows))
	for i, row := range rows {
		usage, recorded := usageByRun[row.ID]
		runs[i] = TaskRun{
			ID:                  row.ID,
			AutomationID:        row.WorkflowID,
			AutomationName:      row.WorkflowName,
			ParentRunID:         row.ParentRunID,
			State:               row.State,
			Result:              row.Result,
			CreatedAt:           row.CreatedAt,
			UpdatedAt:           row.UpdatedAt,
			FinishedAt:          row.FinishedAt,
			ActivityTitle:       row.ActivityTitle,
			ActivityDescription: row.ActivityDescription,
			UsageRecorded:       recorded,
			Usage:               usage,
		}
	}
	if counts == nil {
		counts = []TaskRunCount{}
	}
	return TaskRunHistory{Runs: runs, Counts: counts, HasNextPage: hasNext}, nil
}

func (o *FactoryWorkOrder) listTaskRunCounts(tx *gorm.DB) ([]TaskRunCount, error) {
	querySQL, args, err := o.taskRunQuery(tx)
	if err != nil {
		return nil, err
	}
	var rows []taskRunCountRow
	err = tx.Raw(querySQL+`
		SELECT w.id AS automation_id, w.name AS automation_name,
			COALESCE(r.result, '') AS result, COUNT(*) AS count
		FROM workflow_runs r
		JOIN task_runs t ON r.id = t.id
		JOIN workflows w ON w.id = r.workflow_id
		GROUP BY w.id, w.name, COALESCE(r.result, '')
		ORDER BY w.name ASC, w.id ASC, result ASC
	`, args...).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	counts := make([]TaskRunCount, len(rows))
	for i, row := range rows {
		counts[i] = TaskRunCount{
			AutomationID:   row.AutomationID,
			AutomationName: row.AutomationName,
			Result:         row.Result,
			Count:          row.Count,
		}
	}
	return counts, nil
}

func (o *FactoryWorkOrder) taskRunQuery(tx *gorm.DB) (string, []any, error) {
	subscription, err := o.UsesCodexSubscription(tx)
	if err != nil {
		return "", nil, err
	}
	if subscription {
		return taskRunIDsCTE, []any{o.OrganizationID, o.FactoryID, o.ID}, nil
	}
	return legacyTaskRunIDsCTE, []any{
		o.FactoryID, o.SourceRunID, o.ID,
		o.OrganizationID, o.FactoryID, o.SourceRunID, o.ID, o.ID, o.ID,
		o.ID.String(), o.FactoryID, o.ID.String(), o.ID.String(),
	}, nil
}

type taskRunRow struct {
	ID                  uuid.UUID
	WorkflowID          uuid.UUID
	WorkflowName        string
	ParentRunID         *uuid.UUID
	State               string
	Result              string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	FinishedAt          *time.Time
	ActivityTitle       string
	ActivityDescription string
}

type taskRunCountRow struct {
	AutomationID   uuid.UUID
	AutomationName string
	Result         string
	Count          int64
}

func (o *FactoryWorkOrder) cancelRuns(tx *gorm.DB, update FactoryWorkOrderStatusUpdate) error {
	runs, err := o.ListRuns(tx)
	if err != nil {
		return err
	}
	slices.SortFunc(runs, func(a, b CanvasRun) int {
		return bytes.Compare(a.ID[:], b.ID[:])
	})
	results := make([]*RunCancellationResult, 0, len(runs))
	for i := range runs {
		// An automation can close its own task. Let that run finish its status update.
		if update.Run != nil && runs[i].ID == update.Run.ID {
			continue
		}
		result, err := runs[i].requestRunTermination(tx, update.Actor, "")
		if err != nil {
			return err
		}
		if result == nil || result.Drain == nil {
			continue
		}
		results = append(results, result)
	}
	if update.OnRunsCancelled != nil && len(results) > 0 {
		update.OnRunsCancelled(results)
	}
	return nil
}
