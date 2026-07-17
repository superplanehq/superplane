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
	CanvasRunStateStarted    = "started"
	CanvasRunStateCancelling = "cancelling"
	CanvasRunStateFinished   = "finished"

	CanvasRunResultPassed    = "passed"
	CanvasRunResultFailed    = "failed"
	CanvasRunResultCancelled = "cancelled"

	// Used when locking rows to update non-key columns only, so concurrent child
	// inserts referencing the row via FK are not blocked (PostgreSQL FOR NO KEY UPDATE).
	lockingForUpdateNoKey = "NO KEY UPDATE"
)

type CanvasRun struct {
	ID          uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	WorkflowID  uuid.UUID
	VersionID   uuid.UUID
	State       string
	Result      string
	CancelledAt *time.Time
	CancelledBy *uuid.UUID
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
	FinishedAt  *time.Time
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

	run, err = CreateCanvasRunInTransaction(tx, rootEvent.WorkflowID, CanvasRunStateStarted, "")
	if err != nil {
		return nil, err
	}

	rootEvent.RunID = run.ID
	if err := tx.Model(rootEvent).Update("run_id", run.ID).Error; err != nil {
		return nil, err
	}

	return run, nil
}

func CreateCanvasRunInTransaction(tx *gorm.DB, workflowID uuid.UUID, state, result string) (*CanvasRun, error) {
	liveVersion, err := FindLiveCanvasVersionInTransaction(tx, workflowID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	run := &CanvasRun{
		WorkflowID: workflowID,
		VersionID:  liveVersion.ID,
		State:      state,
		Result:     result,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}

	if state == CanvasRunStateFinished {
		run.FinishedAt = &now
	}

	if err := tx.Create(run).Error; err != nil {
		return nil, err
	}

	return run, nil
}

func ListStartedCanvasRuns(db *gorm.DB, limit int) ([]CanvasRun, error) {
	var runs []CanvasRun
	err := db.
		Where("state = ?", CanvasRunStateStarted).
		Order("updated_at ASC").
		Limit(limit).
		Find(&runs).
		Error
	if err != nil {
		return nil, err
	}

	return runs, nil
}

func ListCancellingCanvasRuns(db *gorm.DB, limit int) ([]CanvasRun, error) {
	var runs []CanvasRun
	err := db.
		Where("state = ?", CanvasRunStateCancelling).
		Order("cancelled_at DESC").
		Limit(limit).
		Find(&runs).
		Error
	if err != nil {
		return nil, err
	}

	return runs, nil
}

type CanvasRunFilters struct {
	States  []string
	Results []string
}

func ListCanvasRuns(workflowID uuid.UUID, limit int, beforeTime *time.Time, filters CanvasRunFilters) ([]CanvasRun, error) {
	return ListCanvasRunsInTransaction(database.Conn(), workflowID, limit, beforeTime, filters)
}

func ListCanvasRunsInTransaction(tx *gorm.DB, workflowID uuid.UUID, limit int, beforeTime *time.Time, filters CanvasRunFilters) ([]CanvasRun, error) {
	var runs []CanvasRun
	query := tx.
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
	return CountCanvasRunsInTransaction(database.Conn(), workflowID, filters)
}

func CountCanvasRunsInTransaction(tx *gorm.DB, workflowID uuid.UUID, filters CanvasRunFilters) (int64, error) {
	var count int64
	query := tx.
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
		return query.Where("(state IN ? OR result IN ?)", filters.States, filters.Results)
	case hasStates:
		return query.Where("state IN ?", filters.States)
	case hasResults:
		return query.Where("result IN ?", filters.Results)
	default:
		return query
	}
}

func ListExecutionsForRunsInTransaction(tx *gorm.DB, workflowID uuid.UUID, runIDs []uuid.UUID) ([]CanvasNodeExecution, error) {
	if len(runIDs) == 0 {
		return []CanvasNodeExecution{}, nil
	}

	var executions []CanvasNodeExecution
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("run_id IN ?", runIDs).
		Order("created_at ASC").
		Find(&executions).
		Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func LockCanvasRunInTransaction(tx *gorm.DB, runID uuid.UUID) (*CanvasRun, error) {
	var run CanvasRun
	err := tx.
		// Run finalization checks for open child work before marking the run
		// finished. Use FOR UPDATE, not FOR NO KEY UPDATE, so concurrent FK
		// inserts for events, queue items, or executions cannot appear between
		// the open-work check and the final state update.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", runID).
		First(&run).
		Error
	if err != nil {
		return nil, err
	}

	return &run, nil
}

type OpenCanvasRunWork struct {
	HasActiveExecutions bool
	HasQueueItems       bool
	HasPendingEvents    bool
}

func (r *CanvasRun) FindOpenWork(tx *gorm.DB) (*OpenCanvasRunWork, error) {
	var result OpenCanvasRunWork
	err := tx.Raw(`
		SELECT
			EXISTS (
				SELECT 1
				FROM workflow_node_executions
				WHERE run_id = ?
				AND state IN (?, ?, ?)
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
		r.ID,
		CanvasNodeExecutionStatePending,
		CanvasNodeExecutionStateStarted,
		CanvasNodeExecutionStateCancelling,
		r.ID,
		r.ID,
		CanvasEventStatePending,
	).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *CanvasRun) CalculateResult(tx *gorm.DB) (string, error) {
	if r.State == CanvasRunStateCancelling {
		return CanvasRunResultCancelled, nil
	}

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
		r.ID,
		CanvasNodeExecutionResultFailed,
		r.ID,
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

type RunCancellationDrainResult struct {
	RequestedExecutionIDs []uuid.UUID
	DeletedQueueItems     []CanvasNodeQueueItem
	SupersededEvents      []CanvasEvent
}

func (r *RunCancellationDrainResult) Empty() bool {
	return len(r.RequestedExecutionIDs) == 0 && len(r.DeletedQueueItems) == 0 && len(r.SupersededEvents) == 0
}

func (r *CanvasRun) DrainForCancellation(tx *gorm.DB, cancelledBy *uuid.UUID) (*RunCancellationDrainResult, error) {
	executions, err := r.ListExecutionsInStates(tx, []string{CanvasNodeExecutionStatePending, CanvasNodeExecutionStateStarted})
	if err != nil {
		return nil, err
	}

	requestedExecutionIDs, err := cancelNodeExecutions(tx, executions, cancelledBy)
	if err != nil {
		return nil, err
	}

	deletedQueueItems, err := r.DeleteQueueItems(tx)
	if err != nil {
		return nil, err
	}

	supersededEvents, err := r.SupersedePendingEvents(tx)
	if err != nil {
		return nil, err
	}

	return &RunCancellationDrainResult{
		RequestedExecutionIDs: requestedExecutionIDs,
		DeletedQueueItems:     deletedQueueItems,
		SupersededEvents:      supersededEvents,
	}, nil
}

func (r *CanvasRun) SupersedePendingEvents(tx *gorm.DB) ([]CanvasEvent, error) {
	var events []CanvasEvent
	err := tx.
		Where("run_id = ?", r.ID).
		Where("state = ?", CanvasEventStatePending).
		Find(&events).
		Error
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return events, nil
	}

	err = tx.
		Model(&CanvasEvent{}).
		Where("run_id = ?", r.ID).
		Where("state = ?", CanvasEventStatePending).
		Update("state", CanvasEventStateRouted).
		Error
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (r *CanvasRun) ListExecutionsInStates(tx *gorm.DB, states []string) ([]CanvasNodeExecution, error) {
	var executions []CanvasNodeExecution
	err := tx.
		Where("workflow_id = ?", r.WorkflowID).
		Where("run_id = ?", r.ID).
		Where("state IN ?", states).
		Find(&executions).
		Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func (r *CanvasRun) DeleteQueueItems(tx *gorm.DB) ([]CanvasNodeQueueItem, error) {
	var deletedQueueItems []CanvasNodeQueueItem
	err := tx.
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}, {Name: "node_id"}, {Name: "run_id"}, {Name: "workflow_id"}}}).
		Where("workflow_id = ?", r.WorkflowID).
		Where("run_id = ?", r.ID).
		Delete(&deletedQueueItems).
		Error
	if err != nil {
		return nil, err
	}

	return deletedQueueItems, nil
}

func (r *CanvasRun) MarkAsCancelling(tx *gorm.DB, cancelledBy *uuid.UUID) error {
	now := time.Now()
	r.State = CanvasRunStateCancelling
	r.CancelledAt = &now
	r.CancelledBy = cancelledBy
	r.UpdatedAt = &now

	return tx.Model(r).
		Updates(map[string]any{
			"state":        CanvasRunStateCancelling,
			"cancelled_at": &now,
			"cancelled_by": cancelledBy,
			"updated_at":   &now,
		}).
		Error
}
