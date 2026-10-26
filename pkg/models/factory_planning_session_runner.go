package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListQueuedRunnerPlanningSessions finds sessions whose displayed queue reason
// can change when another runner task in the organization starts or finishes.
func ListQueuedRunnerPlanningSessions(tx *gorm.DB, organizationID uuid.UUID) ([]FactoryPlanningSession, error) {
	var taskIDs []string
	if err := tx.Model(&RunnerTask{}).
		Where("organization_id = ? AND backend = ? AND state = ?", organizationID, RunnerTaskBackendIntegrated, RunnerTaskStateQueued).
		Pluck("id", &taskIDs).Error; err != nil {
		return nil, err
	}
	if len(taskIDs) == 0 {
		return nil, nil
	}

	var sessions []FactoryPlanningSession
	err := tx.Table("factory_planning_sessions AS session").
		Select("DISTINCT session.*").
		Joins("JOIN workflow_node_executions AS execution ON execution.run_id = session.canvas_run_id").
		Joins("JOIN workflow_node_execution_kvs AS kv ON kv.execution_id = execution.id AND kv.workflow_id = execution.workflow_id AND kv.node_id = execution.node_id AND kv.key = 'task_id'").
		Where("session.organization_id = ? AND session.state = ?", organizationID, PlanningSessionStateRunning).
		Where("kv.value IN ?", taskIDs).
		Scan(&sessions).Error
	return sessions, err
}
