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
	StepIndex      int
	StepName       string
	CreatedAt      time.Time
}

func (i *FactoryWorkOrderQueueItem) Delete(tx *gorm.DB) error {
	return tx.Delete(i).Error
}

// FactoryWorkOrderQueueItemRecord is a queue item joined with its 1-based
// position inside the step's queue, for API serialization.
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
				  AND ahead.step_index = q.step_index
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
func countQueuedFactoryStepItems(tx *gorm.DB, lineID uuid.UUID, stepIndex int) (int64, error) {
	var count int64
	err := tx.
		Model(&FactoryWorkOrderQueueItem{}).
		Where("line_id = ?", lineID).
		Where("step_index = ?", stepIndex).
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
func countActiveFactoryStepExecutions(tx *gorm.DB, lineID uuid.UUID, stepIndex int) (int64, error) {
	var count int64
	err := tx.
		Model(&FactoryWorkOrderExecution{}).
		Where("line_id = ?", lineID).
		Where("step_index = ?", stepIndex).
		Where("status IN ?", []string{
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
	liveSteps := []FactoryLineStep(liveLine.Steps)
	if stepIndex < len(liveSteps) {
		return &liveSteps[stepIndex]
	}

	snapshotSteps := []FactoryLineStep(dispatch.Steps)
	return &snapshotSteps[stepIndex]
}

func findOldestFactoryWorkOrderQueueItem(tx *gorm.DB, lineID uuid.UUID, stepIndex int) (*FactoryWorkOrderQueueItem, error) {
	var item FactoryWorkOrderQueueItem
	err := tx.
		Where("line_id = ? AND step_index = ?", lineID, stepIndex).
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
// cancelled. Called when the work order closes or returns to draft — a
// queued dispatch has no in-flight run, so nothing would ever finish it
// otherwise, and a zombie active dispatch would block re-dispatch after a
// reopen or the open → draft revert. Dispatches with a running step are
// not touched here: the run finalizer cancels them when their run ends.
//
// It holds the admission lock of every involved line while it works, so a
// concurrent admission cannot consume a queue item this close is about to
// drop — without the lock, the close could finish a dispatch whose step
// run had just started.
func (o *FactoryWorkOrder) dropQueuedLineWork(tx *gorm.DB) error {
	var factoryIDs []uuid.UUID
	err := tx.
		Model(&FactoryWorkOrderQueueItem{}).
		Distinct().
		Where("work_order_id = ?", o.ID).
		Order("factory_id").
		Pluck("factory_id", &factoryIDs).
		Error
	if err != nil {
		return err
	}
	if len(factoryIDs) == 0 {
		return nil
	}

	// Factory locks first, then lines, matching EnqueueOrStartStep.
	for _, factoryID := range factoryIDs {
		if _, err := lockFactoryForAdmission(tx, factoryID); err != nil {
			return err
		}
	}

	var lineIDs []uuid.UUID
	err = tx.
		Model(&FactoryWorkOrderQueueItem{}).
		Distinct().
		Where("work_order_id = ?", o.ID).
		Order("line_id").
		Pluck("line_id", &lineIDs).
		Error
	if err != nil {
		return err
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

// AdmitQueuedForStep admits queued dispatches of (lineID, stepIndex) in
// FIFO order, oldest first, while the step and the factory have free
// slots — normally one per finished run, more when a cap was raised
// mid-flight and a finished run reveals the extra capacity. Queue items
// whose work order is no longer open are dropped (their dispatch finishes
// as cancelled) without using a slot. Returns one result per admitted
// dispatch, oldest first; empty when nothing was admitted.
func AdmitQueuedForStep(tx *gorm.DB, lineID uuid.UUID, stepIndex int) ([]*FactoryLineStepResult, error) {
	line, err := lockFactoryAndLineForAdmission(tx, lineID)
	if err != nil {
		return nil, err
	}

	var admitted []*FactoryLineStepResult
	for {
		item, err := findOldestFactoryWorkOrderQueueItem(tx, lineID, stepIndex)
		if err != nil {
			return nil, err
		}
		if item == nil {
			return admitted, nil
		}

		atFactoryCap, err := factoryAtParallelCapacity(tx, line.OrganizationID, line.FactoryID)
		if err != nil {
			return nil, err
		}
		if atFactoryCap {
			return admitted, nil
		}

		dispatch, err := FindWorkOrderLineDispatch(tx, item.LineDispatchID)
		if err != nil {
			return nil, err
		}

		step := admissionStep(line, dispatch, stepIndex)
		active, err := countActiveFactoryStepExecutions(tx, lineID, stepIndex)
		if err != nil {
			return nil, err
		}
		if active >= int64(step.EffectiveMaxParallelism()) {
			return admitted, nil
		}

		result, err := startQueuedFactoryWorkOrder(tx, item, dispatch)
		if err != nil {
			return nil, err
		}
		if result == nil {
			continue
		}
		admitted = append(admitted, result)
	}
}

// AdmitQueuedForFactory admits the oldest queued dispatches in a factory
// whose steps still have room, while the factory has free parallel-task
// slots. Same-step waiters are admitted first by AdmitQueuedForStep; this
// fills leftover factory capacity from other lines.
func AdmitQueuedForFactory(tx *gorm.DB, factoryID uuid.UUID) ([]*FactoryLineStepResult, error) {
	factory, err := lockFactoryForAdmission(tx, factoryID)
	if err != nil {
		return nil, err
	}
	// A deleted factory must not start more work. Its leftover queue
	// items wait for the cleanup worker.
	if factory == nil {
		return nil, nil
	}

	items, err := listFactoryWorkOrderQueueItemsOldestFirst(tx, factoryID)
	if err != nil {
		return nil, err
	}

	blockedSteps := map[factoryStepKey]bool{}
	var admitted []*FactoryLineStepResult
	for i := range items {
		atFactoryCap, err := factoryAtParallelCapacity(tx, factory.OrganizationID, factoryID)
		if err != nil {
			return nil, err
		}
		if atFactoryCap {
			return admitted, nil
		}

		key := factoryStepKey{LineID: items[i].LineID, StepIndex: items[i].StepIndex}
		if blockedSteps[key] {
			continue
		}

		line, err := lockFactoryLineForStepAdmission(tx, items[i].LineID)
		if err != nil {
			return nil, err
		}

		dispatch, err := FindWorkOrderLineDispatch(tx, items[i].LineDispatchID)
		if err != nil {
			return nil, err
		}

		step := admissionStep(line, dispatch, items[i].StepIndex)
		active, err := countActiveFactoryStepExecutions(tx, items[i].LineID, items[i].StepIndex)
		if err != nil {
			return nil, err
		}
		if active >= int64(step.EffectiveMaxParallelism()) {
			blockedSteps[key] = true
			continue
		}

		result, err := startQueuedFactoryWorkOrder(tx, &items[i], dispatch)
		if err != nil {
			return nil, err
		}
		if result == nil {
			continue
		}
		admitted = append(admitted, result)
	}

	return admitted, nil
}

type factoryStepKey struct {
	LineID    uuid.UUID
	StepIndex int
}

func listFactoryWorkOrderQueueItemsOldestFirst(tx *gorm.DB, factoryID uuid.UUID) ([]FactoryWorkOrderQueueItem, error) {
	var items []FactoryWorkOrderQueueItem
	err := tx.
		Where("factory_id = ?", factoryID).
		Order("created_at ASC").
		Order("id ASC").
		Find(&items).
		Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func startQueuedFactoryWorkOrder(
	tx *gorm.DB,
	item *FactoryWorkOrderQueueItem,
	dispatch *FactoryWorkOrderLineDispatch,
) (*FactoryLineStepResult, error) {
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
		return nil, nil
	}

	return dispatch.StartStep(tx, order, item.StepIndex)
}
