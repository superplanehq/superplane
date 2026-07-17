package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CanvasRunStatePending    = "pending"
	CanvasRunStateStarted    = "started"
	CanvasRunStateCancelling = "cancelling"
	CanvasRunStateFinished   = "finished"

	CanvasRunResultPassed    = core.RunResultPassed
	CanvasRunResultFailed    = core.RunResultFailed
	CanvasRunResultCancelled = core.RunResultCancelled

	// Used when locking rows to update non-key columns only, so concurrent child
	// inserts referencing the row via FK are not blocked (PostgreSQL FOR NO KEY UPDATE).
	lockingForUpdateNoKey = "NO KEY UPDATE"
)

type CanvasRun struct {
	ID                uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	WorkflowID        uuid.UUID
	NodeID            string
	VersionID         uuid.UUID
	ParentRunID       *uuid.UUID
	ParentWorkflowID  *uuid.UUID
	ParentExecutionID *uuid.UUID
	Callbacks         datatypes.JSONSlice[core.RunCallback]
	Input             JSONValue
	State             string
	Result            string
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
	CancelledAt       *time.Time
	CancelledBy       *uuid.UUID
	FinishedAt        *time.Time
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

func ListPendingRuns(tx *gorm.DB) ([]CanvasRun, error) {
	var runs []CanvasRun
	err := tx.
		Where("state = ?", CanvasRunStatePending).
		Order("created_at ASC").
		Find(&runs).
		Error
	if err != nil {
		return nil, err
	}
	return runs, nil
}

func (r *CanvasRun) FindTargetNode(tx *gorm.DB) (*CanvasNode, error) {
	node, err := FindCanvasNode(tx, r.WorkflowID, r.NodeID)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (r *CanvasRun) Start(db *gorm.DB) error {
	return db.Model(r).Update("state", CanvasRunStateStarted).Error
}

const maxRunAncestorHops = 256 // safety cap for corrupted parent_run_id cycles in the DB

var (
	// Returned when the same app entrypoint is invoked twice in one chain, e.g.
	// A/run1 -> A/run2 -> A/run1.
	ErrSubRunEntrypointCycle = errors.New("sub-run entrypoint cycle detected")

	// Returned when invoke chains cross back into an app they already visited, e.g.
	// A -> B -> C -> A.
	ErrSubRunWorkflowCycle = errors.New("sub-run workflow cycle detected")

	// Returned when the chain crosses too many app boundaries without cycling, e.g.
	// A -> B -> C -> D -> ... -> Z when Z exceeds the configured limit.
	ErrSubRunCrossWorkflowDepthExceeded = errors.New("sub-run cross-workflow depth exceeded")
)

// ValidateSubRunCreationInTransaction guards sub-run creation before a pending run is
// inserted. All checks walk the parent_run_id chain from the caller's run upward.
//
// Same-app fan-out (forEach, loop, self-invoke into the same workflow) is allowed:
// those create sibling sub-runs and do not deepen the cross-app chain.
//
// Checks (in order):
//  1. Entrypoint cycle — A/run1 -> A/run2 -> A/run1
//  2. Workflow cycle   — A -> B -> C -> A
//  3. Cross-app depth  — A -> B -> C -> D -> ... -> Z (too many app hops)
func ValidateSubRunCreationInTransaction(
	tx *gorm.DB,
	parentRunID uuid.UUID,
	targetWorkflowID uuid.UUID,
	targetNodeID string,
	maxCrossWorkflowDepth int,
) error {
	if parentRunID == uuid.Nil {
		return nil
	}

	ancestors, err := collectRunAncestorsInTransaction(tx, parentRunID)
	if err != nil {
		return err
	}

	// Entrypoint cycle: same app AND same run (or other entry) node already
	// appears in the chain.
	//
	// Catches ping-pong inside one app, e.g. A/run1 -> A/run2 -> A/run1.
	// Does not count forEach/loop siblings: they share a parent run, so earlier
	// siblings are not ancestors of the next create.
	if targetNodeID != "" && entrypointInRuns(ancestors, targetWorkflowID, targetNodeID) {
		return fmt.Errorf("%w: workflow %s node %s already appears in the run chain",
			ErrSubRunEntrypointCycle, targetWorkflowID, targetNodeID)
	}

	// Workflow cycle: crossing back into an app that was already visited after
	// leaving it.
	//
	// Catches A -> B -> C -> A. Skipped when the parent run is already in the
	// target app (same-workflow sub-runs), e.g. forEach in A creating another
	// A/run item.
	if workflowCycleInAncestors(ancestors, targetWorkflowID) {
		return fmt.Errorf("%w: workflow %s already appears in the run chain",
			ErrSubRunWorkflowCycle, targetWorkflowID)
	}

	// Cross-workflow depth: how many times the chain switches apps on the way to
	// the new target. Same-app hops do not increase this count.
	//
	// Catches long chains like A -> B -> C -> D -> ... -> Z when depth reaches
	// maxCrossWorkflowDepth, even if Z is a new app (no cycle).
	// A -> B -> C has depth 3; A -> A -> A (self-invoke) has depth 0.
	depth := crossWorkflowDepthForSubRun(ancestors, targetWorkflowID)
	if depth >= maxCrossWorkflowDepth {
		return fmt.Errorf("%w: depth is %d (max %d)",
			ErrSubRunCrossWorkflowDepthExceeded, depth, maxCrossWorkflowDepth)
	}

	return nil
}

// collectRunAncestorsInTransaction loads the parent run and every ancestor in one
// query, ordered nearest-first: [parent, grandparent, ..., root].
func collectRunAncestorsInTransaction(tx *gorm.DB, runID uuid.UUID) ([]CanvasRun, error) {
	var ancestors []CanvasRun
	err := tx.Raw(`
		WITH RECURSIVE ancestors AS (
			SELECT id, workflow_id, node_id, parent_run_id, 1 AS depth
			FROM workflow_runs
			WHERE id = ?
			UNION ALL
			SELECT wr.id, wr.workflow_id, wr.node_id, wr.parent_run_id, a.depth + 1
			FROM workflow_runs wr
			INNER JOIN ancestors a ON wr.id = a.parent_run_id
			WHERE a.depth < ?
		)
		SELECT id, workflow_id, node_id, parent_run_id
		FROM ancestors
		ORDER BY depth ASC
	`, runID, maxRunAncestorHops).Scan(&ancestors).Error
	if err != nil {
		return nil, err
	}

	if len(ancestors) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	if len(ancestors) == maxRunAncestorHops {
		return nil, fmt.Errorf("run ancestor chain exceeded %d hops", maxRunAncestorHops)
	}

	return ancestors, nil
}

// entrypointInRuns reports whether (workflow, node) already appears in the chain.
// Example match: prior sub-run was A/run1 and the new target is A/run1.
func entrypointInRuns(runs []CanvasRun, workflowID uuid.UUID, nodeID string) bool {
	for _, run := range runs {
		if run.WorkflowID == workflowID && run.NodeID == nodeID {
			return true
		}
	}
	return false
}

// workflowCycleInAncestors detects cross-back into a previously visited app.
//
// Example: chain is C -> B -> A and the new target is A — A is found in older
// ancestors, so this returns true.
//
// ancestors[0] is the parent run. It is excluded when the parent is already in
// the target app, so same-workflow fan-out (forEach in A -> A/item) still works.
func workflowCycleInAncestors(ancestors []CanvasRun, targetWorkflowID uuid.UUID) bool {
	if len(ancestors) == 0 || ancestors[0].WorkflowID == targetWorkflowID {
		return false
	}

	for _, run := range ancestors[1:] {
		if run.WorkflowID == targetWorkflowID {
			return true
		}
	}

	return false
}

// crossWorkflowDepthForSubRun counts app-boundary crossings from the parent run
// up through the root, plus the hop into the new target when that is a different app.
//
// Examples: A -> B -> C targeting D gives depth 4; A -> A -> A (self-invoke) gives 0.
func crossWorkflowDepthForSubRun(ancestors []CanvasRun, targetWorkflowID uuid.UUID) int {
	if len(ancestors) == 0 {
		return 0
	}

	depth := 0
	if ancestors[0].WorkflowID != targetWorkflowID {
		depth++
	}

	for i := 0; i < len(ancestors)-1; i++ {
		if ancestors[i].WorkflowID != ancestors[i+1].WorkflowID {
			depth++
		}
	}

	return depth
}

func ListChildRunsByParentExecutionsInTransaction(
	tx *gorm.DB,
	parentWorkflowID uuid.UUID,
	parentExecutionIDs []uuid.UUID,
) ([]CanvasRun, error) {
	if len(parentExecutionIDs) == 0 {
		return []CanvasRun{}, nil
	}

	var runs []CanvasRun
	err := tx.
		Where("parent_workflow_id = ?", parentWorkflowID).
		Where("parent_execution_id IN ?", parentExecutionIDs).
		Find(&runs).
		Error
	if err != nil {
		return nil, err
	}

	return runs, nil
}

type RunCancellationDrainResult struct {
	RequestedExecutionIDs []uuid.UUID
	DeletedQueueItems     []CanvasNodeQueueItem
	SupersededEvents      []CanvasEvent
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

type canvasRunKey struct {
	WorkflowID uuid.UUID
	RunID      uuid.UUID
}

func FindCanvasRunsByKeysInTransaction(tx *gorm.DB, keys []canvasRunKey) ([]CanvasRun, error) {
	if len(keys) == 0 {
		return []CanvasRun{}, nil
	}

	seen := make(map[canvasRunKey]struct{}, len(keys))
	unique := make([]canvasRunKey, 0, len(keys))
	for _, key := range keys {
		if key.WorkflowID == uuid.Nil || key.RunID == uuid.Nil {
			continue
		}

		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		unique = append(unique, key)
	}

	if len(unique) == 0 {
		return []CanvasRun{}, nil
	}

	query := tx.Model(&CanvasRun{}).Where("1 = 0")
	for _, key := range unique {
		query = query.Or("workflow_id = ? AND id = ?", key.WorkflowID, key.RunID)
	}

	var runs []CanvasRun
	if err := query.Find(&runs).Error; err != nil {
		return nil, err
	}

	return runs, nil
}

func CollectParentRunKeys(runs []CanvasRun) []canvasRunKey {
	keys := make([]canvasRunKey, 0)
	for _, run := range runs {
		if run.ParentRunID == nil || run.ParentWorkflowID == nil {
			continue
		}

		keys = append(keys, canvasRunKey{
			WorkflowID: *run.ParentWorkflowID,
			RunID:      *run.ParentRunID,
		})
	}

	return keys
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
