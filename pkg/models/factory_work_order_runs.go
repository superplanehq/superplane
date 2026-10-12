package models

import (
	"bytes"
	"slices"

	"gorm.io/gorm"
)

// ListRuns includes task automations, line steps, PR activity, and their descendants.
// A creator run shared by other tasks is left out, including as a root for descendants.
func (o *FactoryWorkOrder) ListRuns(tx *gorm.DB) ([]CanvasRun, error) {
	var runs []CanvasRun
	err := tx.Raw(`
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
		)
		SELECT r.* FROM workflow_runs r JOIN task_runs t ON r.id = t.id ORDER BY r.id
	`, o.FactoryID, o.SourceRunID, o.ID,
		o.OrganizationID, o.FactoryID, o.SourceRunID, o.ID, o.ID, o.ID,
		o.ID.String(), o.FactoryID, o.ID.String(), o.ID.String()).Scan(&runs).Error
	return runs, err
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
