package models

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListRuns includes task automations, line steps, PR activity, and their descendants.
// A creator run shared by other tasks is left out, including as a root for descendants.
func (o *FactoryWorkOrder) ListRuns(tx *gorm.DB) ([]CanvasRun, error) {
	var runs []CanvasRun
	err := tx.Raw(`
		WITH RECURSIVE shared_creator AS (
			SELECT source_run_id AS id
			FROM factory_work_orders
			WHERE factory_id = @factory AND source_run_id = @source AND id <> @order AND source_run_id IS NOT NULL
		),
		candidate_runs AS (
			SELECT CAST(@source AS uuid) AS id
			UNION ALL
			SELECT run_id FROM factory_work_order_executions WHERE work_order_id = @order
			UNION ALL
			SELECT canvas_run_id FROM factory_planning_sessions WHERE draft_work_order_id = @order
			UNION ALL
			SELECT a.run_id FROM factory_pull_request_runs a
			JOIN factory_pull_requests p ON p.id = a.pull_request_id WHERE p.work_order_id = @order
			UNION ALL
			SELECT id FROM workflow_runs WHERE input->'work_order'->>'id' = @order_text
			UNION ALL
			SELECT e.run_id FROM workflow_events e
			JOIN workflows w ON w.id = e.workflow_id
			WHERE w.factory_id = @factory AND COALESCE(e.data->'data', e.data)->'workOrder'->>'id' = @order_text
			UNION ALL
			SELECT e.run_id FROM workflow_events e
			JOIN workflows w ON w.id = e.workflow_id
			WHERE w.factory_id = @factory AND COALESCE(e.data->'data', e.data)->'work_order'->>'id' = @order_text
		),
		task_runs AS (
			SELECT r.id FROM workflow_runs r
			JOIN candidate_runs c ON c.id = r.id
			JOIN workflows w ON w.id = r.workflow_id
			WHERE w.organization_id = @organization AND w.factory_id = @factory
			  AND r.id NOT IN (SELECT id FROM shared_creator)
			UNION
			SELECT r.id FROM workflow_runs r
			JOIN task_runs t ON r.parent_run_id = t.id
			WHERE r.id NOT IN (SELECT id FROM shared_creator)
		)
		SELECT r.* FROM workflow_runs r JOIN task_runs t ON r.id = t.id ORDER BY r.id
	`, sql.Named("factory", o.FactoryID), sql.Named("source", o.SourceRunID),
		sql.Named("order", o.ID), sql.Named("organization", o.OrganizationID),
		sql.Named("order_text", o.ID.String())).Scan(&runs).Error
	return runs, err
}

func FinishDeferredRunCancellations(db *gorm.DB, actor *uuid.UUID, runIDs []uuid.UUID, excluded uuid.UUID) ([]*RunCancellationResult, []uuid.UUID, error) {
	pending := uniqueRunIDs(runIDs, excluded)
	if len(pending) == 0 {
		return nil, nil, nil
	}
	slices.SortFunc(pending, func(a, b uuid.UUID) int {
		return bytes.Compare(a[:], b[:])
	})

	var results []*RunCancellationResult
	var remaining []uuid.UUID
	err := db.Transaction(func(tx *gorm.DB) error {
		results = nil
		remaining = nil
		for _, runID := range pending {
			run, err := FindUnscopedCanvasRun(tx, runID)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			if run.State == CanvasRunStateFinished || run.State == CanvasRunStateCancelling {
				continue
			}
			result, err := run.requestRunTermination(tx, actor, "", true)
			if err != nil {
				return err
			}
			if result != nil && result.Deferred {
				remaining = append(remaining, runID)
				continue
			}
			if result != nil && result.Drain != nil {
				results = append(results, result)
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return results, remaining, nil
}

func StopRunForClosedPlanningOrder(tx *gorm.DB, run *CanvasRun) (*RunCancellationResult, error) {
	if run == nil {
		return nil, nil
	}
	loaded, err := FindUnscopedCanvasRun(tx, run.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if loaded.State == CanvasRunStateFinished || loaded.State == CanvasRunStateCancelling {
		return nil, nil
	}
	orderID, err := closedPlanningOrderID(tx, loaded)
	if err != nil || orderID == uuid.Nil {
		return nil, err
	}
	excluded, err := runExcludedByWorkOrderClose(tx, orderID)
	if err != nil {
		return nil, err
	}
	if excluded == loaded.ID {
		return nil, nil
	}
	result, err := loaded.requestCancellationSkippingBusyLock(tx, nil, excluded)
	if err != nil || result == nil || excluded == uuid.Nil {
		return result, err
	}
	result.DeferredRunIDs = slices.DeleteFunc(result.DeferredRunIDs, func(id uuid.UUID) bool {
		return id == excluded
	})
	result.Deferred = len(result.DeferredRunIDs) > 0
	return result, nil
}

func (o *FactoryWorkOrder) cancelRuns(tx *gorm.DB, update FactoryWorkOrderStatusUpdate) error {
	runs, err := o.ListRuns(tx)
	if err != nil {
		return err
	}
	slices.SortFunc(runs, func(a, b CanvasRun) int {
		return bytes.Compare(a.ID[:], b.ID[:])
	})
	results := make([]*RunCancellationResult, 0, len(runs))
	for i := range runs {
		// An automation can close its own task. Let that run finish its status update.
		if update.Run != nil && runs[i].ID == update.Run.ID {
			continue
		}
		result, err := runs[i].requestRunTermination(tx, update.Actor, "", true)
		if err != nil {
			return err
		}
		o.noteDeferredCancellation(result, update)
		if result == nil || result.Drain == nil {
			continue
		}
		results = append(results, result)
	}
	if update.OnRunsCancelled != nil && len(results) > 0 {
		update.OnRunsCancelled(results)
	}
	return nil
}

func (o *FactoryWorkOrder) noteDeferredCancellation(result *RunCancellationResult, update FactoryWorkOrderStatusUpdate) {
	if result == nil || !result.Deferred {
		return
	}
	ids := result.DeferredRunIDs
	if len(ids) == 0 && result.Run != nil {
		ids = []uuid.UUID{result.Run.ID}
	}
	for _, id := range ids {
		if update.Run != nil && id == update.Run.ID {
			continue
		}
		if slices.Contains(o.DeferredRunCancellationIDs, id) {
			continue
		}
		o.DeferredRunCancellationIDs = append(o.DeferredRunCancellationIDs, id)
		o.RunCancellationDeferred = true
	}
}

func uniqueRunIDs(runIDs []uuid.UUID, excluded uuid.UUID) []uuid.UUID {
	pending := make([]uuid.UUID, 0, len(runIDs))
	for _, runID := range runIDs {
		if runID == uuid.Nil || runID == excluded || slices.Contains(pending, runID) {
			continue
		}
		pending = append(pending, runID)
	}
	return pending
}

const closedPlanningOrderAncestorLimit = 32

func closedPlanningOrderID(tx *gorm.DB, run *CanvasRun) (uuid.UUID, error) {
	current := run
	seen := map[uuid.UUID]struct{}{}
	for i := 0; i < closedPlanningOrderAncestorLimit && current != nil; i++ {
		if _, ok := seen[current.ID]; ok {
			return uuid.Nil, nil
		}
		seen[current.ID] = struct{}{}

		session, err := FindPlanningSessionByRun(tx, current.ID)
		if err != nil && !errors.Is(err, ErrFactoryPlanningSessionNotFound) {
			return uuid.Nil, err
		}
		if err == nil && session.DraftWorkOrderID != nil {
			var order FactoryWorkOrder
			err = tx.Select("state").Where("id = ?", *session.DraftWorkOrderID).First(&order).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return uuid.Nil, nil
			}
			if err != nil {
				return uuid.Nil, err
			}
			if order.State != FactoryWorkOrderStateClosed {
				return uuid.Nil, nil
			}
			return *session.DraftWorkOrderID, nil
		}
		if current.ParentRunID == nil {
			return uuid.Nil, nil
		}
		parent, err := FindUnscopedCanvasRun(tx, *current.ParentRunID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, nil
		}
		if err != nil {
			return uuid.Nil, err
		}
		current = parent
	}
	return uuid.Nil, nil
}

func runExcludedByWorkOrderClose(tx *gorm.DB, orderID uuid.UUID) (uuid.UUID, error) {
	var event FactoryWorkOrderEvent
	err := tx.
		Where("work_order_id = ? AND type = ?", orderID, "order.status.updated").
		Where("data->>'toState' = ?", FactoryWorkOrderStateClosed).
		Order("created_at DESC").
		First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, err
	}
	var payload struct {
		Run *struct {
			ID uuid.UUID `json:"id"`
		} `json:"run"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return uuid.Nil, err
	}
	if payload.Run == nil {
		return uuid.Nil, nil
	}
	return payload.Run.ID, nil
}
