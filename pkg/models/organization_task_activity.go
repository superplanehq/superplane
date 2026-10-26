package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// A replacement must wait until cancellation stops the whole previous run
// subtree. Root events preserve task identity when a session detaches its run.
func stoppingOrganizationTasks(tx *gorm.DB, orgID uuid.UUID) *gorm.DB {
	return tx.Raw(`
		WITH RECURSIVE stopping_runs AS (
			SELECT r.id, r.parent_run_id
			FROM workflow_runs r JOIN workflows c ON c.id = r.workflow_id
			WHERE c.organization_id = ? AND r.state = ?
			UNION
			SELECT p.id, p.parent_run_id FROM stopping_runs a
			JOIN workflow_runs p ON p.id = a.parent_run_id
		), stopping_tasks AS (
			SELECT e.work_order_id FROM stopping_runs a
			JOIN factory_work_order_executions e ON e.run_id = a.id
			UNION
			SELECT s.draft_work_order_id FROM stopping_runs a
			JOIN factory_planning_sessions s ON s.canvas_run_id = a.id
			UNION
			SELECT p.work_order_id FROM stopping_runs a
			JOIN factory_pull_request_runs pr ON pr.run_id = a.id
			JOIN factory_pull_requests p ON p.id = pr.pull_request_id
			UNION
			SELECT o.id FROM stopping_runs a
			JOIN workflow_events e ON e.run_id = a.id AND e.execution_id IS NULL
			JOIN workflows c ON c.id = e.workflow_id
			JOIN factory_work_orders o ON o.factory_id = c.factory_id
				AND o.id::text = COALESCE(COALESCE(e.data->'data', e.data)->'workOrder'->>'id', COALESCE(e.data->'data', e.data)->'work_order'->>'id')
		)
		SELECT DISTINCT o.id FROM stopping_tasks s
		JOIN factory_work_orders o ON o.id = s.work_order_id
		WHERE o.organization_id = ?`, orgID, CanvasRunStateCancelling, orgID)
}

func organizationTaskIsStopping(tx *gorm.DB, orgID, workOrderID uuid.UUID) (bool, error) {
	var count int64
	err := tx.Model(&FactoryWorkOrder{}).
		Where("id = ? AND id IN (?)", workOrderID, stoppingOrganizationTasks(tx, orgID)).
		Count(&count).Error
	return count > 0, err
}
