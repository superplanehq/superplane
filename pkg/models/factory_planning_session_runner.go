package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListQueuedRunnerPlanningFactoryIDs groups boards whose displayed queue reason
// can change when organization runner capacity is reserved or released.
func ListQueuedRunnerPlanningFactoryIDs(tx *gorm.DB, organizationID uuid.UUID) ([]uuid.UUID, error) {
	var factoryIDs []uuid.UUID
	err := tx.Table("factory_planning_sessions AS session").
		Distinct("session.factory_id").
		Joins("JOIN workflow_node_executions AS execution ON execution.run_id = session.canvas_run_id").
		Joins("JOIN workflow_node_execution_kvs AS kv ON kv.execution_id = execution.id AND kv.workflow_id = execution.workflow_id AND kv.node_id = execution.node_id AND kv.key = 'task_id'").
		Joins("JOIN runner_tasks AS task ON task.id::text = kv.value AND task.organization_id = session.organization_id").
		Where("session.organization_id = ? AND session.state = ?", organizationID, PlanningSessionStateRunning).
		Where("session.kind = ? AND session.draft_work_order_id IS NOT NULL", PlanningSessionKindWorkOrderAnalysis).
		Where("task.backend = ? AND task.state = ?", RunnerTaskBackendIntegrated, RunnerTaskStateQueued).
		Pluck("session.factory_id", &factoryIDs).Error
	return factoryIDs, err
}
