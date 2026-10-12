package models

import "gorm.io/gorm"

// ListRuns includes task automations, line steps, PR activity, and their descendants.
func (o *FactoryWorkOrder) ListRuns(tx *gorm.DB) ([]CanvasRun, error) {
	var runs []CanvasRun
	err := tx.Raw(`
		WITH RECURSIVE task_runs AS (
			SELECT r.id FROM workflow_runs r
			JOIN workflows w ON w.id = r.workflow_id
			WHERE w.organization_id = ? AND w.factory_id = ? AND (
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
			SELECT r.id FROM workflow_runs r JOIN task_runs t ON r.parent_run_id = t.id
		)
		SELECT r.* FROM workflow_runs r JOIN task_runs t ON r.id = t.id ORDER BY r.id
	`, o.OrganizationID, o.FactoryID, o.SourceRunID, o.ID, o.ID, o.ID,
		o.ID.String(), o.FactoryID, o.ID.String(), o.ID.String()).Scan(&runs).Error
	return runs, err
}

func (o *FactoryWorkOrder) cancelRuns(tx *gorm.DB, update FactoryWorkOrderStatusUpdate) error {
	runs, err := o.ListRuns(tx)
	if err != nil {
		return err
	}
	for i := range runs {
		// An automation can close its own task. Let that run finish its status update.
		if update.Run != nil && runs[i].ID == update.Run.ID {
			continue
		}
		if _, err := runs[i].requestRunTermination(tx, update.Actor, ""); err != nil {
			return err
		}
	}
	return nil
}
