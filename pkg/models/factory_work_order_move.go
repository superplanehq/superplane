package models

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrFactoryTaskMoveInvalid = errors.New("invalid task stage movement")
var ErrFactoryTaskMoveStale = errors.New("calling run no longer owns the current stage")
var ErrFactoryTaskNotWaiting = errors.New("task is not waiting in this stage")

type FactoryTaskMoveResult struct {
	FromColumn     string
	Column         string
	Changed        bool
	Started        []*FactoryLineStepResult
	Cancellations  []*RunCancellationResult
	EnteredColumns []string
}

// MoveFromRun replaces open stage work on the calling run's traversal. The
// caller owns the transaction and publishes its outcomes after commit.
func (o *FactoryWorkOrder) MoveFromRun(tx *gorm.DB, runID uuid.UUID, column string, next bool) (*FactoryTaskMoveResult, error) {
	execution, err := FindWorkOrderExecutionForRun(tx, runID)
	if err != nil {
		return nil, err
	}
	if execution.WorkOrderID != o.ID {
		return nil, ErrFactoryTaskMoveStale
	}

	// Finalizers lock runs before admission. Do not wait on a run whose
	// finalizer could be waiting for our traversal or task lock.
	runs, err := o.ListRuns(tx)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(runs))
	for _, run := range runs {
		ids = append(ids, run.ID)
	}
	var lockedRuns []CanvasRun
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).Where("id IN ?", ids).Order("id").Find(&lockedRuns).Error; err != nil {
		return nil, err
	}
	if _, err := lockFactoryLineForStepAdmission(tx, execution.LineID); err != nil {
		return nil, err
	}
	dispatch, err := LockWorkOrderLineDispatch(tx, execution.LineDispatchID)
	if err != nil {
		return nil, err
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).Where("id = ?", o.ID).First(o).Error; err != nil {
		return nil, err
	}
	execution, err = FindWorkOrderExecutionForRun(tx, runID)
	if err != nil {
		return nil, err
	}
	if !o.IsOpen() || !dispatch.IsActive() || execution.Status == FactoryWorkOrderExecutionStatusFinished || dispatch.CurrentStageKey != dispatch.StepStageKey(execution.StepIndex) {
		return nil, ErrFactoryTaskMoveStale
	}
	caller, err := FindUnscopedCanvasRun(tx, runID)
	if err != nil {
		return nil, err
	}
	if caller.State == CanvasRunStateFinished || caller.State == CanvasRunStateCancelling {
		return nil, ErrFactoryTaskMoveStale
	}
	stage, found := dispatch.StageByKey(column)
	if next {
		stage, found = dispatch.NextStageAfter(dispatch.CurrentStageKey)
	}
	if !found {
		return nil, fmt.Errorf("%w: destination or successor not found", ErrFactoryTaskMoveInvalid)
	}
	result := &FactoryTaskMoveResult{FromColumn: dispatch.CurrentStageKey, Column: stage.Key}
	if stage.Key == dispatch.CurrentStageKey {
		return result, nil
	}
	if stage.Kind != FactoryLineStageBacklog && stage.Kind != FactoryLineStageDone {
		if err := o.RequireExecutionOwner(tx, true); err != nil {
			return nil, err
		}
	}

	// Settle executions even for the calling run. It may finish its response,
	// but it permanently loses authority to advance or move this traversal.
	var openExecutions []FactoryWorkOrderExecution
	if err := tx.Where("line_dispatch_id = ? AND status IN ?", dispatch.ID, []string{FactoryWorkOrderExecutionStatusPending, FactoryWorkOrderExecutionStatusRunning}).Find(&openExecutions).Error; err != nil {
		return nil, err
	}
	for _, superseded := range openExecutions {
		if superseded.RunID == nil || (execution.RunID != nil && *superseded.RunID == *execution.RunID) {
			continue
		}
		run, err := FindUnscopedCanvasRun(tx, *superseded.RunID)
		if err != nil {
			return nil, err
		}
		cancelled, err := run.RequestCancellation(tx, nil)
		if err != nil {
			return nil, err
		}
		result.Cancellations = append(result.Cancellations, cancelled)
	}
	released, err := dispatch.releaseOpenWorkFrom(tx, 0)
	if err != nil {
		return nil, err
	}
	result.Started = released
	if err := o.enterMoveDestination(tx, dispatch, stage, runID, result); err != nil {
		return nil, err
	}
	return result, nil
}

// MoveWaitingFromRun moves a task that waits in stageKey after its stage
// work passed. Runs with no task, such as webhook runs, use it. They can
// move only tasks that wait in their own stage.
func (o *FactoryWorkOrder) MoveWaitingFromRun(tx *gorm.DB, runID uuid.UUID, stageKey, column string, next bool) (*FactoryTaskMoveResult, error) {
	latest, err := o.LatestLineDispatch(tx)
	if errors.Is(err, ErrFactoryWorkOrderLineDispatchNotFound) {
		return nil, ErrFactoryTaskNotWaiting
	}
	if err != nil {
		return nil, err
	}
	if _, err := lockFactoryLineForStepAdmission(tx, latest.LineID); err != nil {
		return nil, err
	}
	dispatch, err := LockWorkOrderLineDispatch(tx, latest.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).Where("id = ?", o.ID).First(o).Error; err != nil {
		return nil, err
	}
	if !o.IsOpen() || dispatch.State != FactoryWorkOrderLineDispatchStateFinished || dispatch.Result != CanvasRunResultPassed || dispatch.CurrentStageKey != stageKey {
		return nil, ErrFactoryTaskNotWaiting
	}
	open, err := dispatch.HasOpenWork(tx)
	if err != nil {
		return nil, err
	}
	if open {
		return nil, ErrFactoryTaskNotWaiting
	}
	stage, found := dispatch.StageByKey(column)
	if next {
		stage, found = dispatch.NextStageAfter(dispatch.CurrentStageKey)
	}
	if !found {
		return nil, fmt.Errorf("%w: destination or successor not found", ErrFactoryTaskMoveInvalid)
	}
	result := &FactoryTaskMoveResult{FromColumn: dispatch.CurrentStageKey, Column: stage.Key}
	if stage.Key == dispatch.CurrentStageKey {
		return result, nil
	}
	if stage.Kind != FactoryLineStageBacklog && stage.Kind != FactoryLineStageDone {
		if err := o.RequireExecutionOwner(tx, true); err != nil {
			return nil, err
		}
	}
	if err := dispatch.Reactivate(tx); err != nil {
		return nil, err
	}
	if err := o.enterMoveDestination(tx, dispatch, stage, runID, result); err != nil {
		return nil, err
	}
	return result, nil
}

// LatestLineDispatch returns the newest line traversal of the task. Its
// current stage is the stage that shows the task on an open task.
func (o *FactoryWorkOrder) LatestLineDispatch(tx *gorm.DB) (*FactoryWorkOrderLineDispatch, error) {
	var latest FactoryWorkOrderLineDispatch
	if err := tx.Where("work_order_id = ?", o.ID).Order("created_at DESC").Order("id DESC").First(&latest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryWorkOrderLineDispatchNotFound
		}
		return nil, err
	}
	return &latest, nil
}

// enterMoveDestination enters stage and follows empty stages and auto
// advance, like the finalizer does after a stage passes.
func (o *FactoryWorkOrder) enterMoveDestination(tx *gorm.DB, dispatch *FactoryWorkOrderLineDispatch, stage FactoryLineStage, runID uuid.UUID, result *FactoryTaskMoveResult) error {
	for {
		if err := dispatch.EnterStage(tx, stage.Key); err != nil {
			return err
		}
		result.EnteredColumns = append(result.EnteredColumns, stage.Key)
		if stage.Kind == FactoryLineStageBacklog || stage.Kind == FactoryLineStageDone {
			state, outcome := FactoryWorkOrderStateDraft, CanvasRunResultCancelled
			if stage.Kind == FactoryLineStageDone {
				state, outcome = FactoryWorkOrderStateClosed, CanvasRunResultPassed
			}
			if err := dispatch.Finish(tx, outcome); err != nil {
				return err
			}
			_, err := o.UpdateStatus(tx, FactoryWorkOrderStatusUpdate{ToState: state, Result: FactoryWorkOrderResultCompleted, Run: &factory.RunRef{ID: runID}, OnRunsCancelled: func(stops []*RunCancellationResult) { result.Cancellations = append(result.Cancellations, stops...) }})
			if err != nil {
				return err
			}
			break
		}
		if index, ok := dispatch.FirstStepIndexForStage(stage.Key); ok {
			for _, step := range dispatch.Steps {
				if step.StageKey != stage.Key {
					continue
				}
				if err := o.ValidateSubscriptionCanvasModels(tx, step.AppID, dispatch.Model); err != nil {
					return err
				}
			}
			started, err := dispatch.StartStage(tx, o, index)
			if err != nil {
				return err
			}
			result.Started = append(result.Started, started...)
			break
		}
		if !stage.AutoAdvanceEnabled() {
			if err := dispatch.Finish(tx, CanvasRunResultPassed); err != nil {
				return err
			}
			break
		}
		successor, ok := dispatch.NextStageAfter(stage.Key)
		if !ok {
			if err := dispatch.Finish(tx, CanvasRunResultPassed); err != nil {
				return err
			}
			break
		}
		stage = successor
	}
	result.Column = dispatch.CurrentStageKey
	result.Changed = true
	return nil
}
