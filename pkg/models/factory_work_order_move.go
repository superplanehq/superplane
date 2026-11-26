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
	for {
		if err := dispatch.EnterStage(tx, stage.Key); err != nil {
			return nil, err
		}
		result.EnteredColumns = append(result.EnteredColumns, stage.Key)
		if stage.Kind == FactoryLineStageBacklog || stage.Kind == FactoryLineStageDone {
			state, outcome := FactoryWorkOrderStateDraft, CanvasRunResultCancelled
			if stage.Kind == FactoryLineStageDone {
				state, outcome = FactoryWorkOrderStateClosed, CanvasRunResultPassed
			}
			if err := dispatch.Finish(tx, outcome); err != nil {
				return nil, err
			}
			_, err := o.UpdateStatus(tx, FactoryWorkOrderStatusUpdate{ToState: state, Result: FactoryWorkOrderResultCompleted, Run: &factory.RunRef{ID: runID}, OnRunsCancelled: func(stops []*RunCancellationResult) { result.Cancellations = append(result.Cancellations, stops...) }})
			if err != nil {
				return nil, err
			}
			break
		}
		if index, ok := dispatch.FirstStepIndexForStage(stage.Key); ok {
			for _, step := range dispatch.Steps {
				if step.StageKey != stage.Key {
					continue
				}
				if err := o.ValidateSubscriptionCanvasModels(tx, step.AppID, dispatch.Model); err != nil {
					return nil, err
				}
			}
			started, err := dispatch.StartStage(tx, o, index)
			if err != nil {
				return nil, err
			}
			result.Started = append(result.Started, started...)
			break
		}
		if !stage.AutoAdvanceEnabled() {
			if err := dispatch.Finish(tx, CanvasRunResultPassed); err != nil {
				return nil, err
			}
			break
		}
		successor, ok := dispatch.NextStageAfter(stage.Key)
		if !ok {
			if err := dispatch.Finish(tx, CanvasRunResultPassed); err != nil {
				return nil, err
			}
			break
		}
		stage = successor
	}
	result.Column = dispatch.CurrentStageKey
	result.Changed = true
	return result, nil
}
