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

	// Used when locking rows to update non-key columns only, so concurrent child
	// inserts referencing the row via FK are not blocked (PostgreSQL FOR NO KEY UPDATE).
	lockingForUpdateNoKey = "NO KEY UPDATE"
)

type CanvasRun struct {
	ID         uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	WorkflowID uuid.UUID
	VersionID  uuid.UUID
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

func ListStartedCanvasRuns(limit int) ([]CanvasRun, error) {
	return ListStartedCanvasRunsInTransaction(database.Conn(), limit)
}

func ListStartedCanvasRunsInTransaction(tx *gorm.DB, limit int) ([]CanvasRun, error) {
	var runs []CanvasRun
	err := tx.
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

type canvasRunCompletionCandidate struct {
	ID                  uuid.UUID
	Result              string
	HasActiveExecutions bool
	HasQueueItems       bool
	HasPendingEvents    bool
}

func FindOpenCanvasRunWorkInTransaction(tx *gorm.DB, runID uuid.UUID) (*OpenCanvasRunWork, error) {
	var result OpenCanvasRunWork
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

func CalculateCanvasRunResultInTransaction(tx *gorm.DB, runID uuid.UUID) (string, error) {
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

func FinishCanvasRunsWithNoOpenWork(tx *gorm.DB, workflowID uuid.UUID, runIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(runIDs) == 0 {
		return []uuid.UUID{}, nil
	}

	candidates, err := listCanvasRunCompletionCandidates(tx, workflowID, runIDs)
	if err != nil {
		return nil, err
	}

	finishedRunIDsByResult := map[string][]uuid.UUID{}
	finishedRunIDs := make([]uuid.UUID, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.hasOpenWork() {
			continue
		}

		finishedRunIDsByResult[candidate.Result] = append(finishedRunIDsByResult[candidate.Result], candidate.ID)
		finishedRunIDs = append(finishedRunIDs, candidate.ID)
	}

	now := time.Now()
	for result, ids := range finishedRunIDsByResult {
		if err := finishCanvasRunsWithResult(tx, workflowID, ids, result, now); err != nil {
			return nil, err
		}
	}

	return finishedRunIDs, nil
}

func (c canvasRunCompletionCandidate) hasOpenWork() bool {
	return c.HasActiveExecutions || c.HasQueueItems || c.HasPendingEvents
}

func listCanvasRunCompletionCandidates(tx *gorm.DB, workflowID uuid.UUID, runIDs []uuid.UUID) ([]canvasRunCompletionCandidate, error) {
	var candidates []canvasRunCompletionCandidate
	err := tx.Raw(`
		SELECT
			workflow_runs.id,
			CASE
				WHEN EXISTS (
					SELECT 1
					FROM workflow_node_executions
					WHERE run_id = workflow_runs.id
					AND result = ?
				) THEN ?
				WHEN EXISTS (
					SELECT 1
					FROM workflow_node_executions
					WHERE run_id = workflow_runs.id
					AND result = ?
				) THEN ?
				ELSE ?
			END AS result,
			EXISTS (
				SELECT 1
				FROM workflow_node_executions
				WHERE run_id = workflow_runs.id
				AND state IN (?, ?)
			) AS has_active_executions,
			EXISTS (
				SELECT 1
				FROM workflow_node_queue_items
				WHERE run_id = workflow_runs.id
			) AS has_queue_items,
			EXISTS (
				SELECT 1
				FROM workflow_events
				WHERE run_id = workflow_runs.id
				AND state = ?
			) AS has_pending_events
		FROM workflow_runs
		WHERE workflow_runs.workflow_id = ?
		AND workflow_runs.id IN ?
		AND workflow_runs.state != ?
		FOR UPDATE
	`,
		CanvasNodeExecutionResultFailed,
		CanvasRunResultFailed,
		CanvasNodeExecutionResultCancelled,
		CanvasRunResultCancelled,
		CanvasRunResultPassed,
		CanvasNodeExecutionStatePending,
		CanvasNodeExecutionStateStarted,
		CanvasEventStatePending,
		workflowID,
		runIDs,
		CanvasRunStateFinished,
	).Scan(&candidates).Error
	if err != nil {
		return nil, err
	}

	return candidates, nil
}

func finishCanvasRunsWithResult(tx *gorm.DB, workflowID uuid.UUID, runIDs []uuid.UUID, result string, now time.Time) error {
	if len(runIDs) == 0 {
		return nil
	}

	return tx.
		Model(&CanvasRun{}).
		Where("workflow_id = ?", workflowID).
		Where("id IN ?", runIDs).
		Updates(map[string]any{
			"state":       CanvasRunStateFinished,
			"result":      result,
			"updated_at":  &now,
			"finished_at": &now,
		}).
		Error
}
