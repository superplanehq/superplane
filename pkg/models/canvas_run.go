package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CanvasRunStateStarted  = "started"
	CanvasRunStateFinished = "finished"

	CanvasRunResultPassed    = "passed"
	CanvasRunResultFailed    = "failed"
	CanvasRunResultCancelled = "cancelled"
)

type CanvasRun struct {
	ID         uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	WorkflowID uuid.UUID
	State      string
	Result     string
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
	FinishedAt *time.Time
}

func (r *CanvasRun) TableName() string {
	return "workflow_runs"
}

func FindCanvasRunInTransaction(tx *gorm.DB, workflowID, runID uuid.UUID) (*CanvasRun, error) {
	var run CanvasRun
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("id = ?", runID).
		First(&run).
		Error
	if err != nil {
		return nil, err
	}

	return &run, nil
}

func FindCanvasRunByRootEventInTransaction(tx *gorm.DB, rootEventID uuid.UUID) (*CanvasRun, error) {
	var run CanvasRun
	err := tx.
		Joins("INNER JOIN workflow_events ON workflow_events.run_id = workflow_runs.id").
		Where("workflow_events.id = ?", rootEventID).
		First(&run).
		Error
	if err != nil {
		return nil, err
	}

	return &run, nil
}

func FindOrCreateCanvasRunForRootEventInTransaction(tx *gorm.DB, rootEvent *CanvasEvent) (*CanvasRun, error) {
	if rootEvent.RunID != uuid.Nil {
		return FindCanvasRunInTransaction(tx, rootEvent.WorkflowID, rootEvent.RunID)
	}

	run, err := FindCanvasRunByRootEventInTransaction(tx, rootEvent.ID)
	if err == nil {
		return run, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	run, err = CreateCanvasRunInTransaction(tx, rootEvent.WorkflowID)
	if err != nil {
		return nil, err
	}

	rootEvent.RunID = run.ID
	if err := tx.Model(rootEvent).Update("run_id", run.ID).Error; err != nil {
		return nil, err
	}

	return run, nil
}

func CreateCanvasRunInTransaction(tx *gorm.DB, workflowID uuid.UUID) (*CanvasRun, error) {
	now := time.Now()
	run := &CanvasRun{
		WorkflowID: workflowID,
		State:      CanvasRunStateStarted,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}

	if err := tx.Create(run).Error; err != nil {
		return nil, err
	}

	return run, nil
}

type CanvasRunFilters struct {
	States  []string
	Results []string
}

func ListCanvasRuns(workflowID uuid.UUID, limit int, beforeTime *time.Time, filters CanvasRunFilters) ([]CanvasRun, error) {
	var runs []CanvasRun
	query := database.Conn().
		Where("workflow_id = ?", workflowID).
		Order("created_at DESC").
		Limit(limit)

	query = applyCanvasRunFilters(query, filters)

	if beforeTime != nil {
		query = query.Where("created_at < ?", beforeTime)
	}

	err := query.Find(&runs).Error
	if err != nil {
		return nil, err
	}

	return runs, nil
}

func CountCanvasRuns(workflowID uuid.UUID, filters CanvasRunFilters) (int64, error) {
	var count int64
	query := database.Conn().
		Model(&CanvasRun{}).
		Where("workflow_id = ?", workflowID)

	query = applyCanvasRunFilters(query, filters)

	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func applyCanvasRunFilters(query *gorm.DB, filters CanvasRunFilters) *gorm.DB {
	hasStates := len(filters.States) > 0
	hasResults := len(filters.Results) > 0

	switch {
	case hasStates && hasResults:
		return query.Where("state IN ? OR result IN ?", filters.States, filters.Results)
	case hasStates:
		return query.Where("state IN ?", filters.States)
	case hasResults:
		return query.Where("result IN ?", filters.Results)
	default:
		return query
	}
}

func ListParentExecutionsForRunsInTransaction(tx *gorm.DB, workflowID uuid.UUID, runIDs []uuid.UUID) ([]CanvasNodeExecution, error) {
	if len(runIDs) == 0 {
		return []CanvasNodeExecution{}, nil
	}

	var executions []CanvasNodeExecution
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("run_id IN ?", runIDs).
		Where("parent_execution_id IS NULL").
		Order("created_at ASC").
		Find(&executions).
		Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func MaybeFinalizeRunInTransaction(tx *gorm.DB, runID uuid.UUID) (bool, error) {
	run, err := lockCanvasRunInTransaction(tx, runID)
	if err != nil {
		return false, err
	}

	if run.State == CanvasRunStateFinished {
		return false, nil
	}

	openWork, err := findOpenCanvasRunWorkInTransaction(tx, runID)
	if err != nil {
		return false, err
	}

	if openWork.HasActiveExecutions || openWork.HasQueueItems || openWork.HasPendingEvents {
		return false, touchCanvasRunInTransaction(tx, run)
	}

	result, err := calculateCanvasRunResultInTransaction(tx, runID)
	if err != nil {
		return false, err
	}

	now := time.Now()
	err = tx.Model(run).
		Updates(map[string]any{
			"state":       CanvasRunStateFinished,
			"result":      result,
			"updated_at":  &now,
			"finished_at": &now,
		}).
		Error
	if err != nil {
		return false, err
	}

	return true, nil
}

func lockCanvasRunInTransaction(tx *gorm.DB, runID uuid.UUID) (*CanvasRun, error) {
	var run CanvasRun
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", runID).
		First(&run).
		Error
	if err != nil {
		return nil, err
	}

	return &run, nil
}

func touchCanvasRunInTransaction(tx *gorm.DB, run *CanvasRun) error {
	now := time.Now()
	return tx.Model(run).Update("updated_at", &now).Error
}

type openCanvasRunWork struct {
	HasActiveExecutions bool
	HasQueueItems       bool
	HasPendingEvents    bool
}

func findOpenCanvasRunWorkInTransaction(tx *gorm.DB, runID uuid.UUID) (*openCanvasRunWork, error) {
	var result openCanvasRunWork
	err := tx.Raw(`
		SELECT
			EXISTS (
				SELECT 1
				FROM workflow_node_executions
				WHERE run_id = ?
				AND state IN (?, ?)
			) AS has_active_executions,
			EXISTS (
				SELECT 1
				FROM workflow_node_queue_items
				WHERE run_id = ?
			) AS has_queue_items,
			EXISTS (
				SELECT 1
				FROM workflow_events
				WHERE run_id = ?
				AND state = ?
			) AS has_pending_events
	`,
		runID,
		CanvasNodeExecutionStatePending,
		CanvasNodeExecutionStateStarted,
		runID,
		runID,
		CanvasEventStatePending,
	).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func calculateCanvasRunResultInTransaction(tx *gorm.DB, runID uuid.UUID) (string, error) {
	var result struct {
		HasFailed    bool
		HasCancelled bool
	}

	err := tx.Raw(`
		SELECT
			EXISTS (
				SELECT 1
				FROM workflow_node_executions
				WHERE run_id = ?
				AND result = ?
			) AS has_failed,
			EXISTS (
				SELECT 1
				FROM workflow_node_executions
				WHERE run_id = ?
				AND result = ?
			) AS has_cancelled
	`,
		runID,
		CanvasNodeExecutionResultFailed,
		runID,
		CanvasNodeExecutionResultCancelled,
	).Scan(&result).Error
	if err != nil {
		return "", err
	}

	if result.HasFailed {
		return CanvasRunResultFailed, nil
	}

	if result.HasCancelled {
		return CanvasRunResultCancelled, nil
	}

	return CanvasRunResultPassed, nil
}
