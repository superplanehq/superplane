package models

import (
	"cmp"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrTaskNoLongerQueued = errors.New("task is no longer queued")

// OrganizationQueuedTask represents either a waiting line step or a task
// root event. IDs identify the specific pending work, so stale cancellation
// requests cannot stop work that has already started or been replaced.
type OrganizationQueuedTask struct {
	ID             uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	RunID          uuid.UUID
	LineDispatchID uuid.UUID
	LineID         uuid.UUID
	StepIndex      int
	CreatedAt      time.Time
	Position       int
	Name           string
	StartsLine     bool
}

func ListOrganizationQueuedTasks(tx *gorm.DB, orgID uuid.UUID) ([]OrganizationQueuedTask, error) {
	var queued []OrganizationQueuedTask
	err := tx.Table("factory_work_order_queue_items AS q").Select("q.*, q.step_name AS name, NOT EXISTS (SELECT 1 FROM factory_work_order_executions e WHERE e.line_dispatch_id = q.line_dispatch_id) AS starts_line").
		Joins("JOIN factory_work_order_line_dispatches d ON d.id = q.line_dispatch_id").
		Joins("JOIN factory_work_orders o ON o.id = q.work_order_id").
		Joins("JOIN factories f ON f.id = q.factory_id AND f.deleted_at IS NULL").
		Where("q.organization_id = ? AND d.state = ? AND o.state IN ?", orgID, FactoryWorkOrderLineDispatchStateActive, []string{FactoryWorkOrderStateDraft, FactoryWorkOrderStateOpen}).
		Scan(&queued).Error
	if err != nil {
		return nil, err
	}
	analysis, err := listQueuedRootTasks(tx, orgID)
	if err != nil {
		return nil, err
	}
	queued = append(queued, analysis...)
	slices.SortFunc(queued, func(a, b OrganizationQueuedTask) int {
		if order := a.CreatedAt.Compare(b.CreatedAt); order != 0 {
			return order
		}
		return cmp.Compare(a.ID.String(), b.ID.String())
	})
	positions := make(map[uuid.UUID]int)
	for i := range queued {
		if positions[queued[i].WorkOrderID] == 0 {
			positions[queued[i].WorkOrderID] = len(positions) + 1
		}
		queued[i].Position = positions[queued[i].WorkOrderID]
	}
	return queued, nil
}

func (o *FactoryWorkOrder) CancelQueuedTask(tx *gorm.DB, queueID uuid.UUID) (*RunCancellationResult, error) {
	if err := LockOrganizationTaskAdmission(tx, o.OrganizationID); err != nil {
		return nil, err
	}
	if err := o.LockForUpdate(tx); err != nil {
		return nil, err
	}
	queued, err := ListOrganizationQueuedTasks(tx, o.OrganizationID)
	if err != nil {
		return nil, err
	}
	for _, task := range queued {
		if task.ID != queueID || task.FactoryID != o.FactoryID || task.WorkOrderID != o.ID {
			continue
		}
		if task.LineDispatchID != uuid.Nil {
			item := &FactoryWorkOrderQueueItem{ID: task.ID, LineDispatchID: task.LineDispatchID}
			dispatch, err := FindWorkOrderLineDispatch(tx, task.LineDispatchID)
			if err != nil {
				return nil, err
			}
			if err := discardQueuedTask(tx, item, dispatch, CanvasRunResultCancelled); err != nil {
				return nil, err
			}
			if !task.StartsLine {
				return nil, nil
			}
			active, err := ListActiveOrganizationTaskIDs(tx, o.OrganizationID)
			if err != nil || slices.Contains(active, o.ID) {
				return nil, err
			}
			_, err = o.UpdateStatus(tx, FactoryWorkOrderStatusUpdate{ToState: FactoryWorkOrderStateDraft, SkipSame: true})
			return nil, err
		}
		run, err := FindUnscopedCanvasRun(tx, task.RunID)
		if err != nil {
			return nil, err
		}
		if err := o.endQueuedAnalysisSession(tx, run); err != nil {
			return nil, err
		}
		return run.RequestCancellation(tx, nil)
	}
	return nil, ErrTaskNoLongerQueued
}

func (o *FactoryWorkOrder) endQueuedAnalysisSession(tx *gorm.DB, run *CanvasRun) error {
	session, err := FindPlanningSessionByDraftWorkOrder(tx, o.OrganizationID, o.FactoryID, o.ID)
	if errors.Is(err, ErrFactoryPlanningSessionNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if session.CanvasID == nil || *session.CanvasID != run.WorkflowID {
		return nil
	}
	if session.CanvasRunID != nil && *session.CanvasRunID != run.ID {
		return nil
	}
	return session.End(tx)
}

func listQueuedRootTasks(tx *gorm.DB, orgID uuid.UUID) ([]OrganizationQueuedTask, error) {
	var queued []OrganizationQueuedTask
	query := tx.Table("workflow_events AS e").
		Select("e.id, c.factory_id, o.id AS work_order_id, e.run_id, e.created_at, c.name").
		Joins("JOIN workflows c ON c.id = e.workflow_id AND c.deleted_at IS NULL").
		Joins("JOIN factories f ON f.id = c.factory_id AND f.deleted_at IS NULL").
		Joins("JOIN workflow_runs r ON r.id = e.run_id").
		Joins("JOIN factory_work_orders o ON o.id::text = COALESCE(COALESCE(e.data->'data', e.data)->'workOrder'->>'id', COALESCE(e.data->'data', e.data)->'work_order'->>'id') AND o.factory_id = c.factory_id AND o.organization_id = c.organization_id").
		Where("c.organization_id = ? AND e.execution_id IS NULL AND e.state = ?", orgID, CanvasEventStatePending).
		Where("o.state IN ? AND r.state IN ?", []string{FactoryWorkOrderStateDraft, FactoryWorkOrderStateOpen}, []string{CanvasRunStateStarted, CanvasRunStatePending}).
		Where("NOT EXISTS (SELECT 1 FROM factory_work_order_executions x WHERE x.run_id = r.id)")
	err := query.Order("e.created_at, e.id").Scan(&queued).Error
	return queued, err
}

// Locally full steps and stopping tasks do not prevent other eligible tasks
// from using an organization slot. Cards retain the full queue order.
func nextQueuedOrganizationTask(tx *gorm.DB, orgID uuid.UUID) (*OrganizationQueuedTask, error) {
	queued, err := ListOrganizationQueuedTasks(tx, orgID)
	if err != nil {
		return nil, err
	}
	var stopping []uuid.UUID
	if err := stoppingOrganizationTasks(tx, orgID).Scan(&stopping).Error; err != nil {
		return nil, err
	}
	reserved, err := mergeReservedWorkOrderIDs(tx, orgID)
	if err != nil {
		return nil, err
	}
	for i := range queued {
		task := &queued[i]
		if slices.Contains(stopping, task.WorkOrderID) || slices.Contains(reserved, task.WorkOrderID) {
			continue
		}
		if task.RunID != uuid.Nil {
			return task, nil
		}
		item := &FactoryWorkOrderQueueItem{
			ID: task.ID, OrganizationID: orgID, FactoryID: task.FactoryID,
			WorkOrderID: task.WorkOrderID, LineDispatchID: task.LineDispatchID,
			LineID: task.LineID, StepIndex: task.StepIndex,
		}
		dispatch, order, line, err := queuedTaskCandidate(tx, item)
		if err != nil {
			return nil, err
		}
		if dispatch == nil || order == nil || line == nil || task.StepIndex < 0 || task.StepIndex >= len(dispatch.Steps) {
			continue
		}
		active, err := countActiveFactoryStepExecutions(tx, task.LineID, task.StepIndex)
		if err != nil {
			return nil, err
		}
		if active < int64(admissionStep(line, dispatch, task.StepIndex).EffectiveMaxParallelism()) {
			return task, nil
		}
	}
	return nil, nil
}
