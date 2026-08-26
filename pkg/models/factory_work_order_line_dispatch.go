package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryWorkOrderLineDispatchStateActive   = "active"
	FactoryWorkOrderLineDispatchStateFinished = "finished"
)

var (
	ErrFactoryWorkOrderLineDispatchNotFound = errors.New("factory work order line dispatch not found")

	// ErrFactoryWorkOrderLineDispatchActive replaces
	// ErrFactoryWorkOrderExecutionActive as the sentinel returned by the
	// dispatch guard and the open -> draft revert guard.
	ErrFactoryWorkOrderLineDispatchActive = errors.New("work order already has an active line dispatch")
)

// FactoryWorkOrderLineDispatch is one traversal of a work order through a
// factory line: "this work order's pass through Line 1". It snapshots the
// line's steps at dispatch time and carries the traversal's state/result,
// so both are read directly instead of being derived from child step
// executions every time they're needed. See issue #6737.
type FactoryWorkOrderLineDispatch struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	LineID         uuid.UUID
	LineName       string
	Steps          datatypes.JSONSlice[FactoryLineStep]
	State          string
	Result         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	FinishedAt     *time.Time
}

func (FactoryWorkOrderLineDispatch) TableName() string {
	return "factory_work_order_line_dispatches"
}

func (l *FactoryWorkOrderLineDispatch) IsActive() bool {
	return l.State == FactoryWorkOrderLineDispatchStateActive
}

func (l *FactoryWorkOrderLineDispatch) Ref() *factory.LineRef {
	return &factory.LineRef{ID: l.LineID, Name: l.LineName}
}

// Dispatch creates the line dispatch for order's traversal of l — snapshotting
// l's current name/steps — and starts (or queues) step 0 inside it. Both
// writes happen in the caller's transaction so a partial dispatch (parent
// created, step 0 failed) can never be observed.
func (l *FactoryLine) Dispatch(tx *gorm.DB, order *FactoryWorkOrder) (*FactoryWorkOrderLineDispatch, *FactoryLineStepResult, error) {
	return l.DispatchFrom(tx, order, 0)
}

// DispatchFrom creates a line dispatch and starts (or queues) the step at
// startIndex. Rerun from the start uses 0. Rerun this step uses the
// current step.
func (l *FactoryLine) DispatchFrom(tx *gorm.DB, order *FactoryWorkOrder, startIndex int) (*FactoryWorkOrderLineDispatch, *FactoryLineStepResult, error) {
	if len(l.Steps) == 0 {
		return nil, nil, ErrFactoryLineHasNoSteps
	}
	if startIndex < 0 || startIndex >= len(l.Steps) {
		return nil, nil, ErrFactoryLineStepOutOfRange
	}

	now := time.Now()
	dispatch := &FactoryWorkOrderLineDispatch{
		ID:             uuid.New(),
		OrganizationID: l.OrganizationID,
		FactoryID:      l.FactoryID,
		WorkOrderID:    order.ID,
		LineID:         l.ID,
		LineName:       l.Name,
		Steps:          l.Steps,
		State:          FactoryWorkOrderLineDispatchStateActive,
		Result:         "",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(dispatch).Error; err != nil {
		return nil, nil, err
	}

	result, err := dispatch.EnqueueOrStartStep(tx, order, startIndex)
	if err != nil {
		return nil, nil, err
	}

	return dispatch, result, nil
}

// EnqueueOrStartStep starts the step at stepIndex when it has a free slot,
// or queues the dispatch for admission when the step is at its
// maxParallelism. The step's queue is FIFO: when other dispatches already
// wait for the step, the newcomer joins the back of the queue even if a
// slot is free (a raised maxParallelism can leave free slots behind queued
// work). It takes the line's admission lock, so concurrent decisions for
// the same line cannot both see the last free slot.
func (l *FactoryWorkOrderLineDispatch) EnqueueOrStartStep(tx *gorm.DB, order *FactoryWorkOrder, stepIndex int) (*FactoryLineStepResult, error) {
	steps := []FactoryLineStep(l.Steps)
	if stepIndex < 0 || stepIndex >= len(steps) {
		return nil, fmt.Errorf("step index %d out of range", stepIndex)
	}

	line, err := lockFactoryLineForStepAdmission(tx, l.LineID)
	if err != nil {
		return nil, err
	}

	queued, err := countQueuedFactoryStepItems(tx, l.LineID, stepIndex)
	if err != nil {
		return nil, err
	}
	if queued > 0 {
		return l.enqueueStep(tx, order, stepIndex)
	}

	step := admissionStep(line, l, stepIndex)
	active, err := countActiveFactoryStepExecutions(tx, l.LineID, stepIndex)
	if err != nil {
		return nil, err
	}
	if active >= int64(step.EffectiveMaxParallelism()) {
		return l.enqueueStep(tx, order, stepIndex)
	}

	return l.StartStep(tx, order, stepIndex)
}

// enqueueStep parks the dispatch in the step's queue and records the
// `step.execution.queued` timeline event. The step name is the app
// (canvas) name, captured when the dispatch queues — the same name the
// execution captures when the step starts.
func (l *FactoryWorkOrderLineDispatch) enqueueStep(tx *gorm.DB, order *FactoryWorkOrder, stepIndex int) (*FactoryLineStepResult, error) {
	step := []FactoryLineStep(l.Steps)[stepIndex]

	canvas, err := FindCanvasInTransaction(tx, l.OrganizationID, step.AppID)
	if err != nil {
		return nil, err
	}

	item := &FactoryWorkOrderQueueItem{
		ID:             uuid.New(),
		OrganizationID: l.OrganizationID,
		FactoryID:      l.FactoryID,
		WorkOrderID:    order.ID,
		LineID:         l.LineID,
		LineDispatchID: l.ID,
		StepIndex:      stepIndex,
		StepName:       canvas.Name,
		CreatedAt:      time.Now(),
	}

	if err := tx.Clauses(clause.Returning{}).Create(item).Error; err != nil {
		return nil, err
	}

	if err := l.RecordStepExecutionQueued(tx, order, item, &step); err != nil {
		return nil, err
	}

	return &FactoryLineStepResult{QueueItem: item}, nil
}

// StartStep launches the step at stepIndex in the dispatch's steps
// snapshot — not the line's live steps — so a line edited mid-traversal
// can't change what runs next for an in-flight dispatch.
func (l *FactoryWorkOrderLineDispatch) StartStep(tx *gorm.DB, order *FactoryWorkOrder, stepIndex int) (*FactoryLineStepResult, error) {
	steps := []FactoryLineStep(l.Steps)
	if stepIndex < 0 || stepIndex >= len(steps) {
		return nil, fmt.Errorf("step index %d out of range", stepIndex)
	}

	step := steps[stepIndex]

	node, err := FindCanvasNode(tx, step.AppID, step.Entrypoint)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("entrypoint %q not found", step.Entrypoint)
		}
		return nil, err
	}

	ref := node.Ref.Data()
	if ref.Trigger == nil || ref.Trigger.Name != onRunTriggerName {
		return nil, ErrFactoryLineStepNotOnRun
	}

	liveVersion, err := FindLiveCanvasVersionInTransaction(tx, step.AppID)
	if err != nil {
		return nil, err
	}

	canvas, err := FindCanvasInTransaction(tx, l.OrganizationID, step.AppID)
	if err != nil {
		return nil, err
	}

	runInput, err := factoryWorkOrderRunInput(tx, order)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	run := &CanvasRun{
		ID:         uuid.New(),
		WorkflowID: step.AppID,
		NodeID:     step.Entrypoint,
		VersionID:  liveVersion.ID,
		Callbacks: datatypes.JSONSlice[core.RunCallback]{
			{
				When: core.RunCallbackWhenPending,
				On:   core.RunCallbackOnEntry,
				Hook: "onMessage",
			},
		},
		Input:     NewJSONValue(runInput),
		State:     CanvasRunStatePending,
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	if err := tx.Create(run).Error; err != nil {
		return nil, err
	}

	execution := &FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: l.OrganizationID,
		FactoryID:      l.FactoryID,
		WorkOrderID:    order.ID,
		LineID:         l.LineID,
		LineDispatchID: l.ID,
		StepIndex:      stepIndex,
		StepName:       canvas.Name,
		RunID:          &run.ID,
		Status:         FactoryWorkOrderExecutionStatusPending,
		Result:         "",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(execution).Error; err != nil {
		return nil, err
	}

	if err := l.RecordStepExecutionCreated(tx, order, execution, run); err != nil {
		return nil, err
	}

	return &FactoryLineStepResult{
		Run:       run,
		Execution: execution,
	}, nil
}

func (l *FactoryWorkOrderLineDispatch) RecordStepExecutionQueued(
	tx *gorm.DB,
	order *FactoryWorkOrder,
	item *FactoryWorkOrderQueueItem,
	step *FactoryLineStep,
) error {
	data := factory.LineStepExecutionQueued{
		StepName: item.StepName,
		Order:    order.Ref(),
		Line:     l.Ref(),
		App:      &factory.AppRef{ID: step.AppID, Name: item.StepName},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	event := &FactoryWorkOrderEvent{
		ID:          uuid.New(),
		WorkOrderID: order.ID,
		Type:        factory.EventTypeLineStepExecutionQueued,
		Data:        datatypes.JSON(jsonData),
		CreatedAt:   time.Now(),
	}

	return tx.Create(event).Error
}

func (l *FactoryWorkOrderLineDispatch) RecordStepExecutionCreated(
	tx *gorm.DB,
	order *FactoryWorkOrder,
	execution *FactoryWorkOrderExecution,
	run *CanvasRun,
) error {
	data := factory.LineStepExecutionCreated{
		StepName: execution.StepName,
		Order:    order.Ref(),
		Line:     l.Ref(),
		App:      &factory.AppRef{ID: run.WorkflowID, Name: execution.StepName},
		Run:      &factory.RunRef{ID: run.ID, State: run.State},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	event := &FactoryWorkOrderEvent{
		ID:          uuid.New(),
		WorkOrderID: order.ID,
		Type:        factory.EventTypeLineStepExecutionCreated,
		Data:        datatypes.JSON(jsonData),
		CreatedAt:   time.Now(),
	}

	return tx.Create(event).Error
}

// Reactivate opens a finished traversal again so a later step can start
// on the same history. No-op if the dispatch is already active.
func (l *FactoryWorkOrderLineDispatch) Reactivate(tx *gorm.DB) error {
	if l.State != FactoryWorkOrderLineDispatchStateFinished {
		return nil
	}

	now := time.Now()
	l.State = FactoryWorkOrderLineDispatchStateActive
	l.Result = ""
	l.FinishedAt = nil
	l.UpdatedAt = now

	return tx.Model(l).Updates(map[string]any{
		"state":       l.State,
		"result":      l.Result,
		"finished_at": nil,
		"updated_at":  l.UpdatedAt,
	}).Error
}

func (l *FactoryWorkOrderLineDispatch) Finish(tx *gorm.DB, result string) error {
	if l.State == FactoryWorkOrderLineDispatchStateFinished {
		return nil
	}

	now := time.Now()
	l.State = FactoryWorkOrderLineDispatchStateFinished
	l.Result = result
	l.UpdatedAt = now
	l.FinishedAt = &now

	return tx.Model(l).Updates(map[string]any{
		"state":       l.State,
		"result":      l.Result,
		"updated_at":  l.UpdatedAt,
		"finished_at": l.FinishedAt,
	}).Error
}

// AbandonActiveLineDispatch finishes the current traversal as cancelled
// so a new dispatch can start. No-op when no active dispatch exists.
func (o *FactoryWorkOrder) AbandonActiveLineDispatch(tx *gorm.DB) error {
	if err := o.dropQueuedLineWork(tx); err != nil {
		return err
	}

	dispatch, err := o.FindActiveLineDispatch(tx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	return dispatch.Finish(tx, CanvasRunResultCancelled)
}

// RetryLineStep starts stepIndex again on a traversal that already ran
// earlier steps. It does not open a new dispatch, so earlier steps stay
// on the same history.
func (o *FactoryWorkOrder) RetryLineStep(tx *gorm.DB, line *FactoryLine, stepIndex int) (*FactoryLineStepResult, error) {
	if stepIndex < 0 {
		return nil, ErrFactoryLineStepOutOfRange
	}

	byOrder, err := ListWorkOrderLineDispatchesByWorkOrderIDs(tx, []uuid.UUID{o.ID})
	if err != nil {
		return nil, err
	}

	var onLine []FactoryWorkOrderLineDispatchRecord
	for _, record := range byOrder[o.ID] {
		if record.LineID == line.ID {
			onLine = append(onLine, record)
		}
	}

	target := pickDispatchForStepRetry(onLine, stepIndex)
	if target == nil {
		_, result, err := line.DispatchFrom(tx, o, stepIndex)
		return result, err
	}

	active, err := o.FindActiveLineDispatch(tx)
	if err == nil && active.ID != target.ID {
		if err := o.AbandonActiveLineDispatch(tx); err != nil {
			return nil, err
		}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := target.Reactivate(tx); err != nil {
		return nil, err
	}

	if err := target.settleOpenWorkFrom(tx, stepIndex); err != nil {
		return nil, err
	}

	for index := stepIndex; index < len(target.Steps); index++ {
		if _, err := AdmitQueuedForStep(tx, line.ID, index); err != nil {
			return nil, err
		}
	}

	return target.EnqueueOrStartStep(tx, o, stepIndex)
}

func (l *FactoryWorkOrderLineDispatch) settleOpenWorkFrom(tx *gorm.DB, stepIndex int) error {
	if err := tx.
		Where("line_dispatch_id = ? AND step_index >= ?", l.ID, stepIndex).
		Delete(&FactoryWorkOrderQueueItem{}).Error; err != nil {
		return err
	}

	var executions []FactoryWorkOrderExecution
	err := tx.
		Where("line_dispatch_id = ? AND step_index >= ?", l.ID, stepIndex).
		Where("status IN ?", []string{
			FactoryWorkOrderExecutionStatusPending,
			FactoryWorkOrderExecutionStatusRunning,
		}).
		Find(&executions).Error
	if err != nil {
		return err
	}

	for i := range executions {
		if err := executions[i].MarkFinished(tx, CanvasRunResultCancelled); err != nil {
			return err
		}
	}

	return nil
}

func pickDispatchForStepRetry(records []FactoryWorkOrderLineDispatchRecord, stepIndex int) *FactoryWorkOrderLineDispatch {
	for i := len(records) - 1; i >= 0; i-- {
		if dispatchHasEarlierStep(records[i], stepIndex) {
			dispatch := records[i].FactoryWorkOrderLineDispatch
			return &dispatch
		}
	}
	if len(records) == 0 {
		return nil
	}
	dispatch := records[len(records)-1].FactoryWorkOrderLineDispatch
	return &dispatch
}

func dispatchHasEarlierStep(record FactoryWorkOrderLineDispatchRecord, stepIndex int) bool {
	if stepIndex <= 0 {
		return true
	}
	for _, execution := range record.Executions {
		if execution.StepIndex < stepIndex {
			return true
		}
	}
	return false
}

// FindActiveLineDispatch is a single indexed lookup ("does an active
// traversal exist for this work order") replacing a scan over child
// executions by status.
func (o *FactoryWorkOrder) FindActiveLineDispatch(tx *gorm.DB) (*FactoryWorkOrderLineDispatch, error) {
	var dispatch FactoryWorkOrderLineDispatch
	err := tx.
		Where("work_order_id = ?", o.ID).
		Where("state = ?", FactoryWorkOrderLineDispatchStateActive).
		First(&dispatch).
		Error
	if err != nil {
		return nil, err
	}

	return &dispatch, nil
}

// ensureNoActiveLineDispatch returns ErrFactoryWorkOrderLineDispatchActive
// if the work order has an active traversal, nil otherwise.
func (o *FactoryWorkOrder) ensureNoActiveLineDispatch(tx *gorm.DB) error {
	_, err := o.FindActiveLineDispatch(tx)
	if err == nil {
		return ErrFactoryWorkOrderLineDispatchActive
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}

func FindWorkOrderLineDispatch(tx *gorm.DB, id uuid.UUID) (*FactoryWorkOrderLineDispatch, error) {
	var dispatch FactoryWorkOrderLineDispatch
	err := tx.Where("id = ?", id).First(&dispatch).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryWorkOrderLineDispatchNotFound
		}
		return nil, err
	}

	return &dispatch, nil
}

// FactoryWorkOrderLineDispatchRecord is a line dispatch with its child step
// executions — and, when the dispatch is waiting for step admission, its
// queue item — preloaded, as used by the serialization layer.
type FactoryWorkOrderLineDispatchRecord struct {
	FactoryWorkOrderLineDispatch
	Executions []FactoryWorkOrderExecutionRecord
	QueueItem  *FactoryWorkOrderQueueItemRecord
}

// ListWorkOrderLineDispatchesByWorkOrderIDs bulk-loads line dispatches (with
// their step executions) for the given work orders, keyed by work order id.
// Dispatches for a given order are ordered oldest-first, matching how
// re-dispatches of the same line should render as separate, ordered
// traversals.
func ListWorkOrderLineDispatchesByWorkOrderIDs(
	tx *gorm.DB,
	workOrderIDs []uuid.UUID,
) (map[uuid.UUID][]FactoryWorkOrderLineDispatchRecord, error) {
	result := make(map[uuid.UUID][]FactoryWorkOrderLineDispatchRecord, len(workOrderIDs))
	if len(workOrderIDs) == 0 {
		return result, nil
	}

	var dispatches []FactoryWorkOrderLineDispatch
	err := tx.
		Where("work_order_id IN ?", workOrderIDs).
		Order("created_at ASC").
		Order("id ASC").
		Find(&dispatches).
		Error
	if err != nil {
		return nil, err
	}

	dispatchIDs := make([]uuid.UUID, len(dispatches))
	for i := range dispatches {
		dispatchIDs[i] = dispatches[i].ID
	}

	executionsByDispatchID, err := ListFactoryWorkOrderExecutionsByLineDispatchIDs(tx, dispatchIDs)
	if err != nil {
		return nil, err
	}

	queueItemsByDispatchID, err := ListFactoryWorkOrderQueueItemsByLineDispatchIDs(tx, dispatchIDs)
	if err != nil {
		return nil, err
	}

	for i := range dispatches {
		dispatch := dispatches[i]
		record := FactoryWorkOrderLineDispatchRecord{
			FactoryWorkOrderLineDispatch: dispatch,
			Executions:                   executionsByDispatchID[dispatch.ID],
		}
		if queueItem, ok := queueItemsByDispatchID[dispatch.ID]; ok {
			record.QueueItem = &queueItem
		}
		result[dispatch.WorkOrderID] = append(result[dispatch.WorkOrderID], record)
	}

	return result, nil
}
