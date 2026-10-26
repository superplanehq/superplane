package models

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const RunnerTaskQueueReasonParallelismLimit = "parallelism_limit"

// RunnerTaskExecutionStatus describes the runner behind a node execution.
// A started node can still have a queued runner task.
type RunnerTaskExecutionStatus struct {
	ExecutionID uuid.UUID
	State       string
	QueueReason string
}

func ListRunnerTaskExecutionStatuses(tx *gorm.DB, executionIDs []uuid.UUID) (map[uuid.UUID]RunnerTaskExecutionStatus, error) {
	statuses := make(map[uuid.UUID]RunnerTaskExecutionStatus, len(executionIDs))
	if len(executionIDs) == 0 {
		return statuses, nil
	}

	type record struct {
		ExecutionID    uuid.UUID
		OrganizationID uuid.UUID
		Value          string
	}
	var records []record
	err := tx.Table("workflow_node_execution_kvs AS kv").
		Select("DISTINCT ON (kv.execution_id) kv.execution_id, canvas.organization_id, kv.value").
		Joins("JOIN workflow_node_executions AS execution ON execution.id = kv.execution_id AND execution.workflow_id = kv.workflow_id AND execution.node_id = kv.node_id").
		Joins("JOIN workflows AS canvas ON canvas.id = execution.workflow_id").
		Where("kv.execution_id IN ? AND kv.key = ?", executionIDs, "task_id").
		Order("kv.execution_id, kv.created_at DESC, kv.id DESC").
		Scan(&records).Error
	if err != nil {
		return nil, err
	}

	taskIDs := make([]uuid.UUID, 0, len(records))
	taskByExecution := make(map[uuid.UUID]uuid.UUID, len(records))
	for _, record := range records {
		taskID, err := uuid.Parse(record.Value)
		if err != nil {
			continue
		}
		taskIDs = append(taskIDs, taskID)
		taskByExecution[record.ExecutionID] = taskID
	}
	if len(taskIDs) == 0 {
		return statuses, nil
	}
	var tasks []RunnerTask
	if err := tx.Select("id", "organization_id", "state").
		Where("id IN ? AND backend = ?", taskIDs, RunnerTaskBackendIntegrated).
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	tasksByID := make(map[uuid.UUID]RunnerTask, len(tasks))
	for _, task := range tasks {
		tasksByID[task.ID] = task
	}

	limitedOrganizations := make(map[uuid.UUID]bool)
	for _, record := range records {
		task, ok := tasksByID[taskByExecution[record.ExecutionID]]
		if !ok || task.OrganizationID != record.OrganizationID {
			continue
		}
		status := RunnerTaskExecutionStatus{ExecutionID: record.ExecutionID, State: task.State}
		if task.State == RunnerTaskStateQueued {
			limited, checked := limitedOrganizations[record.OrganizationID]
			if !checked {
				err := checkOrganizationRunnerCapacity(tx, record.OrganizationID)
				if err != nil && !errors.Is(err, ErrOrganizationRunnerCapacity) {
					return nil, err
				}
				limited = errors.Is(err, ErrOrganizationRunnerCapacity)
				limitedOrganizations[record.OrganizationID] = limited
			}
			if limited {
				status.QueueReason = RunnerTaskQueueReasonParallelismLimit
			}
		}
		statuses[record.ExecutionID] = status
	}
	return statuses, nil
}
