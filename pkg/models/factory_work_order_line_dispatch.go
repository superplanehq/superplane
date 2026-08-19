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
	if len(l.Steps) == 0 {
		return nil, nil, ErrFactoryLineHasNoSteps
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

	result, err := dispatch.EnqueueOrStartStep(tx, order, 0)
	if err != nil {
		return nil, nil, err
	}

	return dispatch, result, nil
}

// EnqueueOrStartStep starts the step at stepIndex when it has a free slot,
// or queues the dispatch for admission when the step is at its
// maxParallelism. It takes the line's admission lock, so concurrent
// decisions for the same line cannot both see the last free slot.
func (l *FactoryWorkOrderLineDispatch) EnqueueOrStartStep(tx *gorm.DB, order *FactoryWorkOrder, stepIndex int) (*FactoryLineStepResult, error) {
	steps := []FactoryLineStep(l.Steps)
	if stepIndex < 0 || stepIndex >= len(steps) {
		return nil, fmt.Errorf("step index %d out of range", stepIndex)
	}

	line, err := lockFactoryLineForStepAdmission(tx, l.LineID)
	if err != nil {
		return nil, err
	}

	step := admissionStep(line, l, stepIndex)
	if !step.UnlimitedParallelism() {
		active, err := countActiveFactoryStepExecutions(tx, l.LineID, stepIndex)
		if err != nil {
			return nil, err
		}
		if active >= int64(step.EffectiveMaxParallelism()) {
			return l.enqueueStep(tx, order, stepIndex)
		}
	}

	return l.StartStep(tx, order, stepIndex)
}

// enqueueStep parks the dispatch in the step's queue and records the
// `step.execution.queued` timeline event. The step name comes from the
// dispatch's snapshot — that is what will run on admission.
func (l *FactoryWorkOrderLineDispatch) enqueueStep(tx *gorm.DB, order *FactoryWorkOrder, stepIndex int) (*FactoryLineStepResult, error) {
	step := []FactoryLineStep(l.Steps)[stepIndex]

	item := &FactoryWorkOrderQueueItem{
		ID:             uuid.New(),
		OrganizationID: l.OrganizationID,
		FactoryID:      l.FactoryID,
		WorkOrderID:    order.ID,
		LineID:         l.LineID,
		LineDispatchID: l.ID,
		StepIndex:      stepIndex,
		StepName:       step.Name,
		CreatedAt:      time.Now(),
	}

	if err := tx.Clauses(clause.Returning{}).Create(item).Error; err != nil {
		return nil, err
	}

	if err := l.RecordStepExecutionQueued(tx, order, &step); err != nil {
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
		StepName:       step.Name,
		RunID:          run.ID,
		Status:         FactoryWorkOrderExecutionStatusPending,
		Result:         "",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(execution).Error; err != nil {
		return nil, err
	}

	if err := l.RecordStepExecutionCreated(tx, order, execution, &step, run); err != nil {
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
	step *FactoryLineStep,
) error {
	data := factory.LineStepExecutionQueued{
		StepName: step.Name,
		Order:    order.Ref(),
		Line:     l.Ref(),
		App:      &factory.AppRef{ID: step.AppID},
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
	step *FactoryLineStep,
	run *CanvasRun,
) error {
	data := factory.LineStepExecutionCreated{
		StepName: step.Name,
		Order:    order.Ref(),
		Line:     l.Ref(),
		App:      &factory.AppRef{ID: run.WorkflowID},
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

// Finish transitions the dispatch to its terminal state. Called from the
// advancement path when a step run fails, is cancelled, or the last step in
// the snapshot passes. No-op if already finished.
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
