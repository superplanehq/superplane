package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Active task identity is shared by line steps, analysis, and pull-request
// automations. Descendant runs retain that identity after their parent ends.
// Pending root events do not reserve capacity until routing admits them.
func ListActiveOrganizationTaskIDs(tx *gorm.DB, orgID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := tx.Raw(`
		WITH RECURSIVE active_runs AS (
			SELECT r.id AS active_id, r.id, r.parent_run_id
			FROM workflow_runs r JOIN workflows c ON c.id = r.workflow_id
			WHERE c.organization_id = ? AND r.state IN ?
			UNION
			SELECT a.active_id, p.id, p.parent_run_id
			FROM active_runs a JOIN workflow_runs p ON p.id = a.parent_run_id
		), active_tasks AS (
			SELECT e.work_order_id
			FROM factory_work_order_executions e
			WHERE e.organization_id = ? AND e.status IN ?
			UNION
			SELECT e.work_order_id
			FROM factory_work_order_executions e
			JOIN workflow_runs r ON r.id = e.run_id
			WHERE e.organization_id = ? AND r.state <> ?
			UNION
			SELECT e.work_order_id FROM active_runs a
			JOIN factory_work_order_executions e ON e.run_id = a.id
			WHERE a.active_id <> a.id
			UNION
			SELECT s.draft_work_order_id FROM active_runs a
			JOIN factory_planning_sessions s ON s.canvas_run_id = a.id
			UNION
			SELECT p.work_order_id FROM active_runs a
			JOIN factory_pull_request_runs pr ON pr.run_id = a.id
			JOIN factory_pull_requests p ON p.id = pr.pull_request_id
			UNION
			SELECT o.id FROM active_runs a
			JOIN workflow_events e ON e.run_id = a.id AND e.execution_id IS NULL AND e.state = ?
			JOIN workflows c ON c.id = e.workflow_id
			JOIN factory_work_orders o ON o.factory_id = c.factory_id
				AND o.id::text = COALESCE(COALESCE(e.data->'data', e.data)->'workOrder'->>'id', COALESCE(e.data->'data', e.data)->'work_order'->>'id')
			WHERE NOT EXISTS (SELECT 1 FROM factory_work_order_executions x WHERE x.run_id = a.id)
				AND (EXISTS (SELECT 1 FROM workflow_node_queue_items q WHERE q.run_id = a.id)
					OR EXISTS (SELECT 1 FROM workflow_node_executions x WHERE x.run_id = a.id))
		)
		SELECT DISTINCT o.id FROM active_tasks a
		JOIN factory_work_orders o ON o.id = a.work_order_id
		WHERE o.organization_id = ?`,
		orgID, []string{CanvasRunStatePending, CanvasRunStateStarted, CanvasRunStateCancelling},
		orgID, []string{FactoryWorkOrderExecutionStatusPending, FactoryWorkOrderExecutionStatusRunning},
		orgID, CanvasRunStateFinished,
		CanvasEventStateRouted, orgID).Scan(&ids).Error
	return ids, err
}
