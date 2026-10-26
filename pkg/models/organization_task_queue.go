package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrRunnerJobNoLongerQueued = errors.New("runner job is no longer queued")

// OrganizationQueuedRunnerJob describes an individual job. Positions include
// jobs without a factory task association and never group sibling jobs.
type OrganizationQueuedRunnerJob struct {
	ID             uuid.UUID
	WorkOrderID    uuid.UUID
	AutomationID   uuid.UUID
	AutomationName string
	NodeID         string
	NodeName       string
	RunID          uuid.UUID
	ExecutionID    uuid.UUID
	QueuedAt       time.Time
	Position       int64
}

func ListOrganizationQueuedRunnerJobs(tx *gorm.DB, orgID uuid.UUID) ([]OrganizationQueuedRunnerJob, error) {
	var rows []struct {
		OrganizationQueuedRunnerJob
		ParentRunID *uuid.UUID
	}
	err := tx.Table("runner_tasks AS t").Select(`
        t.id, t.queued_at,
        ROW_NUMBER() OVER (ORDER BY t.created_at, t.id) AS position,
        e.workflow_id AS automation_id, c.name AS automation_name,
        e.node_id, n.name AS node_name, e.run_id, e.id AS execution_id, r.parent_run_id
    `).
		Joins(`LEFT JOIN LATERAL (
            SELECT x.* FROM workflow_node_executions x
            JOIN workflows owner ON owner.id = x.workflow_id AND owner.organization_id = t.organization_id
            JOIN workflow_node_execution_kvs kv ON kv.execution_id = x.id
              AND kv.workflow_id = x.workflow_id AND kv.node_id = x.node_id
            WHERE kv.key = 'task_id' AND kv.value = t.id::text
            ORDER BY x.created_at, x.id LIMIT 1
        ) e ON TRUE`).
		Joins("LEFT JOIN workflow_runs r ON r.id = e.run_id").
		Joins("LEFT JOIN workflows c ON c.id = e.workflow_id AND c.organization_id = t.organization_id").
		Joins("LEFT JOIN workflow_nodes n ON n.workflow_id = e.workflow_id AND n.node_id = e.node_id").
		Where("t.organization_id = ? AND t.backend = ? AND t.state = ? AND t.runner_id IS NULL", orgID, RunnerTaskBackendIntegrated, RunnerTaskStateQueued).
		Order("t.created_at, t.id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	jobs := make([]OrganizationQueuedRunnerJob, 0, len(rows))
	runsByCanvas := map[uuid.UUID][]CanvasRun{}
	for _, row := range rows {
		jobs = append(jobs, row.OrganizationQueuedRunnerJob)
		if row.AutomationID == uuid.Nil || row.RunID == uuid.Nil {
			continue
		}
		runsByCanvas[row.AutomationID] = append(runsByCanvas[row.AutomationID], CanvasRun{ID: row.RunID, ParentRunID: row.ParentRunID})
	}
	labels := map[uuid.UUID]CanvasRunTaskLabel{}
	for canvasID, runs := range runsByCanvas {
		canvas, err := FindUnscopedCanvasInTransaction(tx, canvasID)
		if err != nil {
			return nil, err
		}
		resolved, err := ResolveCanvasRunTasks(tx, canvas, runs)
		if err != nil {
			return nil, err
		}
		for runID, label := range resolved {
			labels[runID] = label
		}
	}
	for i := range jobs {
		jobs[i].WorkOrderID = labels[jobs[i].RunID].ID
	}
	return jobs, nil
}

// CancelQueuedRunnerJob cancels exactly one certainly-unassigned job. The
// organization lock serializes this check with every reservation writer.
func (o *FactoryWorkOrder) CancelQueuedRunnerJob(tx *gorm.DB, jobID uuid.UUID) (*CanvasNodeExecution, error) {
	if err := LockOrganizationRunnerAdmission(tx, o.OrganizationID); err != nil {
		return nil, err
	}
	task, err := FindRunnerTask(tx.Clauses(clause.Locking{Strength: "UPDATE"}), jobID)
	if errors.Is(err, ErrRunnerTaskNotFound) {
		return nil, ErrRunnerJobNoLongerQueued
	}
	if err != nil {
		return nil, err
	}
	if task.OrganizationID != o.OrganizationID || task.Backend != RunnerTaskBackendIntegrated || task.State != RunnerTaskStateQueued || task.RunnerID != nil {
		return nil, ErrRunnerJobNoLongerQueued
	}
	execution, err := task.FindExecution(tx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRunnerJobNoLongerQueued
	}
	if err != nil {
		return nil, err
	}
	canvas, err := FindUnscopedCanvasInTransaction(tx, execution.WorkflowID)
	if err != nil {
		return nil, err
	}
	run, err := FindUnscopedCanvasRun(tx, execution.RunID)
	if err != nil {
		return nil, err
	}
	labels, err := ResolveCanvasRunTasks(tx, canvas, []CanvasRun{*run})
	if err != nil {
		return nil, err
	}
	if canvas.OrganizationID != o.OrganizationID || labels[run.ID].ID != o.ID {
		return nil, ErrRunnerJobNoLongerQueued
	}
	now := time.Now()
	if err := task.RequestCancel(tx, now); err != nil {
		return nil, err
	}
	// The existing job-specific poll finalizes execution and credential
	// cleanup. Execution cancellation could stop a replacement job.
	return execution, nil
}

// ListRunnerJobsForExecutions returns the durable job recorded by each execution.
func ListRunnerJobsForExecutions(tx *gorm.DB, executionIDs []uuid.UUID) (map[uuid.UUID]RunnerTask, error) {
	result := map[uuid.UUID]RunnerTask{}
	if len(executionIDs) == 0 {
		return result, nil
	}
	var rows []struct {
		RunnerTask
		ExecutionID uuid.UUID
	}
	err := tx.Table("runner_tasks AS t").Select("DISTINCT ON (kv.execution_id) t.*, kv.execution_id").
		Joins("JOIN workflow_node_execution_kvs kv ON kv.key = 'task_id' AND kv.value = t.id::text").
		Where("kv.execution_id IN ? AND t.backend = ?", executionIDs, RunnerTaskBackendIntegrated).
		Order("kv.execution_id, kv.created_at DESC, kv.id DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ExecutionID] = row.RunnerTask
	}
	return result, nil
}
