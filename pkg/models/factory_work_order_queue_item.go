package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FactoryWorkOrderQueueItem is a line dispatch waiting for admission into
// a step that is at its maxParallelism. No run or execution exists for it
// yet; both are created when the step admits the dispatch, and the queue
// item is deleted. A dispatch waits at no more than one step at a time
// (line_dispatch_id is unique).
type FactoryWorkOrderQueueItem struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	LineID         uuid.UUID
	LineDispatchID uuid.UUID
	AppID          uuid.UUID
	StageKey       string
	StepIndex      int
	StepName       string
	CreatedAt      time.Time
}

func (i *FactoryWorkOrderQueueItem) Delete(tx *gorm.DB) error {
	return tx.Delete(i).Error
}

// FactoryWorkOrderQueueItemRecord is a queue item joined with its 1-based
// position inside the automation's queue, for API serialization.
type FactoryWorkOrderQueueItemRecord struct {
	FactoryWorkOrderQueueItem
	Position int
}

// ListFactoryWorkOrderQueueItemsByLineDispatchIDs bulk-loads queue items
// for the given line dispatches, keyed by line_dispatch_id. Position is
// the item's 1-based place in its step's FIFO queue across all work
// orders of the line, so the UI can render "3rd in queue".
func ListFactoryWorkOrderQueueItemsByLineDispatchIDs(
	tx *gorm.DB,
	lineDispatchIDs []uuid.UUID,
) (map[uuid.UUID]FactoryWorkOrderQueueItemRecord, error) {
	result := make(map[uuid.UUID]FactoryWorkOrderQueueItemRecord, len(lineDispatchIDs))
	if len(lineDispatchIDs) == 0 {
		return result, nil
	}

	var records []FactoryWorkOrderQueueItemRecord
	err := tx.
		Table("factory_work_order_queue_items AS q").
		Select(`
			q.*,
			(
				SELECT COUNT(*)
				FROM factory_work_order_queue_items ahead
				WHERE ahead.line_id = q.line_id
				  AND ahead.app_id = q.app_id
				  AND ahead.stage_key = q.stage_key
				  AND (ahead.created_at, ahead.id) < (q.created_at, q.id)
			) + 1 AS position
		`).
		Where("q.line_dispatch_id IN ?", lineDispatchIDs).
		Scan(&records).
		Error
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		result[record.LineDispatchID] = record
	}

	return result, nil
}

// lockFactoryLineForStepAdmission serializes step admission decisions for
// one line: every path that counts a step's active executions and then
// starts or queues work must hold this lock, or two concurrent decisions
// could both see a free slot and exceed the step's maxParallelism.
func lockFactoryLineForStepAdmission(tx *gorm.DB, lineID uuid.UUID) (*FactoryLine, error) {
	var line FactoryLine
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", lineID).
		First(&line).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryLineNotFound
		}
		return nil, err
	}

	return &line, nil
}

// countQueuedFactoryStepItems counts the dispatches waiting in the step's
// queue across all work orders of the line.
func countQueuedFactoryStepItems(tx *gorm.DB, lineID, appID uuid.UUID, stageKey string) (int64, error) {
	var count int64
	err := tx.
		Model(&FactoryWorkOrderQueueItem{}).
		Where("line_id = ?", lineID).
		Where("app_id = ?", appID).
		Where("stage_key = ?", stageKey).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// countActiveFactoryStepExecutions counts the step's in-flight runs across
// all work orders and dispatches of the line — the number a step's
// maxParallelism caps.
func countActiveFactoryStepExecutions(tx *gorm.DB, lineID, appID uuid.UUID, stageKey string) (int64, error) {
	var count int64
	err := tx.
		Model(&FactoryWorkOrderExecution{}).
		Joins("JOIN workflow_runs ON workflow_runs.id = factory_work_order_executions.run_id").
		Joins("JOIN factory_work_order_line_dispatches ON factory_work_order_line_dispatches.id = factory_work_order_executions.line_dispatch_id").
		Where("factory_work_order_executions.line_id = ?", lineID).
		Where("workflow_runs.workflow_id = ?", appID).
		Where("COALESCE(factory_work_order_line_dispatches.steps -> factory_work_order_executions.step_index ->> 'stage_key', '') = ?", stageKey).
		Where("factory_work_order_executions.status IN ?", []string{
			FactoryWorkOrderExecutionStatusPending,
			FactoryWorkOrderExecutionStatusRunning,
		}).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// admissionStep resolves which step definition governs admission — what
// the dispatch is snapshotted to run stays fixed, but the capacity cap is
// a live line resource. The live line wins when it still has a step at
// this index, so editing a step's maxParallelism takes effect for
// in-flight traversals immediately; the dispatch snapshot is the fallback
// when the live line got shorter.
func admissionStep(liveLine *FactoryLine, dispatch *FactoryWorkOrderLineDispatch, stepIndex int) *FactoryLineStep {
	snapshotSteps := []FactoryLineStep(dispatch.Steps)
	snapshot := &snapshotSteps[stepIndex]
	liveSteps := []FactoryLineStep(liveLine.Steps)
	if snapshot.StageKey != "" {
		for _, stage := range liveLine.StagesValue() {
			if stage.Key == snapshot.StageKey && stage.StepIndex != nil && *stage.StepIndex < len(liveSteps) {
				return &liveSteps[*stage.StepIndex]
			}
		}
	}
	if stepIndex < len(liveSteps) && liveSteps[stepIndex].AppID == snapshot.AppID {
		return &liveSteps[stepIndex]
	}
	for index := range liveSteps {
		if liveSteps[index].AppID == snapshot.AppID {
			return &liveSteps[index]
		}
	}
	return snapshot
}

func findOldestFactoryWorkOrderQueueItem(tx *gorm.DB, lineID, appID uuid.UUID, stageKey string) (*FactoryWorkOrderQueueItem, error) {
	var item FactoryWorkOrderQueueItem
	err := tx.
		Where("line_id = ? AND app_id = ? AND stage_key = ?", lineID, appID, stageKey).
		Order("created_at ASC").
		Order("id ASC").
		First(&item).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &item, nil
}

// dropQueuedLineWork abandons a traversal that is waiting in a step's
// queue: the queue item is deleted and its dispatch finishes as
// cancelled. Called when the work order closes — a queued dispatch has no
// in-flight run, so nothing would ever finish it otherwise, and a zombie
// active dispatch would block re-dispatch after a reopen. Dispatches with
// a running step are not touched here: the run finalizer cancels them
// when their run ends.
//
// It holds the admission lock of every involved line while it works, so a
// concurrent admission cannot consume a queue item this close is about to
// drop — without the lock, the close could finish a dispatch whose step
// run had just started.
func (o *FactoryWorkOrder) dropQueuedLineWork(tx *gorm.DB) error {
	var lineIDs []uuid.UUID
	err := tx.
		Model(&FactoryWorkOrderQueueItem{}).
		Distinct().
		Where("work_order_id = ?", o.ID).
		Order("line_id").
		Pluck("line_id", &lineIDs).
		Error
	if err != nil {
		return err
	}
	if len(lineIDs) == 0 {
		return nil
	}

	// Sorted lock order, so two closes over the same lines cannot deadlock.
	for _, lineID := range lineIDs {
		if _, err := lockFactoryLineForStepAdmission(tx, lineID); err != nil {
			return err
		}
	}

	// Re-read under the locks: an item seen above may have been admitted
	// (and its row deleted) before the lock was acquired.
	var items []FactoryWorkOrderQueueItem
	if err := tx.Where("work_order_id = ?", o.ID).Find(&items).Error; err != nil {
		return err
	}

	for i := range items {
		dispatch, err := FindWorkOrderLineDispatch(tx, items[i].LineDispatchID)
		if err != nil {
			return err
		}

		if err := items[i].Delete(tx); err != nil {
			return err
		}

		if err := dispatch.Finish(tx, CanvasRunResultCancelled); err != nil {
			return err
		}
	}

	return nil
}

// AdmitQueuedForStep admits queued dispatches of (lineID, appID) in
// FIFO order, oldest first, while the step has free slots — normally one
// per finished run, more when the step's maxParallelism was raised
// mid-flight and a finished run reveals the extra capacity. Queue items
// whose work order is no longer open are dropped (their dispatch finishes
// as cancelled) without using a slot. Returns one result per admitted
// dispatch, oldest first; empty when nothing was admitted.
func AdmitQueuedForStep(tx *gorm.DB, lineID, appID uuid.UUID, stageKey string) ([]*FactoryLineStepResult, error) {
	line, err := lockFactoryLineForStepAdmission(tx, lineID)
	if err != nil {
		return nil, err
	}

	var admitted []*FactoryLineStepResult
	for {
		item, err := findOldestFactoryWorkOrderQueueItem(tx, lineID, appID, stageKey)
		if err != nil {
			return nil, err
		}
		if item == nil {
			return admitted, nil
		}

		dispatch, err := FindWorkOrderLineDispatch(tx, item.LineDispatchID)
		if err != nil {
			return nil, err
		}

		step := admissionStep(line, dispatch, item.StepIndex)
		active, err := countActiveFactoryStepExecutions(tx, lineID, appID, stageKey)
		if err != nil {
			return nil, err
		}
		if active >= int64(step.EffectiveMaxParallelism()) {
			return admitted, nil
		}

		if err := item.Delete(tx); err != nil {
			return nil, err
		}

		f, err := FindFactory(tx, item.OrganizationID, item.FactoryID)
		if err != nil {
			return nil, err
		}

		order, err := f.FindWorkOrder(tx, item.WorkOrderID)
		if err != nil {
			return nil, err
		}

		// Safety net: the close path drops queued work, but an order that
		// slipped through must not start a run. Abandon its traversal and
		// try the next item in the queue.
		if !order.IsOpen() {
			if err := dispatch.Finish(tx, CanvasRunResultCancelled); err != nil {
				return nil, err
			}
			continue
		}

		result, err := dispatch.StartStep(tx, order, item.StepIndex)
		if err != nil {
			return nil, err
		}
		admitted = append(admitted, result)
	}
}
