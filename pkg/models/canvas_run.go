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
	if rootEvent.RunID != nil {
		return FindCanvasRunInTransaction(tx, rootEvent.WorkflowID, *rootEvent.RunID)
	}

	run, err := FindCanvasRunByRootEventInTransaction(tx, rootEvent.ID)
	if err == nil {
		return run, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now()
	run = &CanvasRun{
		WorkflowID: rootEvent.WorkflowID,
		State:      CanvasRunStateStarted,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}

	if err := tx.Create(run).Error; err != nil {
		return nil, err
	}

	rootEvent.RunID = &run.ID
	if err := tx.Model(rootEvent).Update("run_id", run.ID).Error; err != nil {
		return nil, err
	}

	return run, nil
}

func ListCanvasRuns(workflowID uuid.UUID, limit int, beforeTime *time.Time) ([]CanvasRun, error) {
	var runs []CanvasRun
	query := database.Conn().
		Where("workflow_id = ?", workflowID).
		Order("created_at DESC").
		Limit(limit)

	if beforeTime != nil {
		query = query.Where("created_at < ?", beforeTime)
	}

	err := query.Find(&runs).Error
	if err != nil {
		return nil, err
	}

	return runs, nil
}

func CountCanvasRuns(workflowID uuid.UUID) (int64, error) {
	var count int64
	err := database.Conn().
		Model(&CanvasRun{}).
		Where("workflow_id = ?", workflowID).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
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
	var failedCount int64
	err := tx.
		Model(&CanvasNodeExecution{}).
		Where("run_id = ?", runID).
		Where("result = ?", CanvasNodeExecutionResultFailed).
		Count(&failedCount).
		Error
	if err != nil {
		return "", err
	}

	if failedCount > 0 {
		return CanvasRunResultFailed, nil
	}

	var cancelledCount int64
	err = tx.
		Model(&CanvasNodeExecution{}).
		Where("run_id = ?", runID).
		Where("result = ?", CanvasNodeExecutionResultCancelled).
		Count(&cancelledCount).
		Error
	if err != nil {
		return "", err
	}

	if cancelledCount > 0 {
		return CanvasRunResultCancelled, nil
	}

	return CanvasRunResultPassed, nil
}
