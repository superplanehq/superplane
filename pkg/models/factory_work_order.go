package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	FactoryWorkOrderStateDraft  = "draft"
	FactoryWorkOrderStateOpen   = "open"
	FactoryWorkOrderStateClosed = "closed"

	FactoryWorkOrderResultCompleted = "completed"
	FactoryWorkOrderResultRejected  = "rejected"
	FactoryWorkOrderResultFailed    = "failed"
)

var (
	ErrFactoryWorkOrderNotFound     = errors.New("factory work order not found")
	ErrFactoryWorkOrderInvalidState = errors.New("invalid work order state transition")
)

var (
	factoryWorkOrderStates = []string{
		FactoryWorkOrderStateDraft,
		FactoryWorkOrderStateOpen,
		FactoryWorkOrderStateClosed,
	}

	factoryWorkOrderCloseResults = []string{
		FactoryWorkOrderResultCompleted,
		FactoryWorkOrderResultRejected,
		FactoryWorkOrderResultFailed,
	}

	//
	// Allowed transitions between states. The updater always emits an
	// `order.status.updated` event; explicit lifecycle events (`order.opened`,
	// `order.closed`) remain for backwards compatibility with the timeline.
	//
	// Dispatching a `draft` order promotes it to `open` (see
	// TransitionOnDispatch). `open → draft` supports "back to draft" edits.
	// `closed → open` is the reopen path.
	//
	factoryWorkOrderAllowedTransitions = map[string][]string{
		FactoryWorkOrderStateDraft:  {FactoryWorkOrderStateOpen},
		FactoryWorkOrderStateOpen:   {FactoryWorkOrderStateClosed, FactoryWorkOrderStateDraft},
		FactoryWorkOrderStateClosed: {FactoryWorkOrderStateOpen},
	}
)

type FactoryWorkOrder struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	Title          string
	Description    string
	State          string
	Result         string
	CreatedByID    *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	//
	// StateUpdatedAt is bumped only when `UpdateStatus` moves the order
	// through a lifecycle transition. It stays put on assignee changes,
	// comments, and artifact writes. The display-status logic uses it as
	// the fence for "current attempt" so a reopened order isn't tagged
	// `failed` by executions from the previous attempt, and re-assigning
	// a failed order doesn't hide the failure.
	//
	StateUpdatedAt time.Time

	CreatedBy *User                      `gorm:"foreignKey:CreatedByID"`
	Assignees []FactoryWorkOrderAssignee `gorm:"foreignKey:WorkOrderID"`
}

func (FactoryWorkOrder) TableName() string {
	return "factory_work_orders"
}

func (o *FactoryWorkOrder) IsOpen() bool {
	return o.State == FactoryWorkOrderStateOpen
}

func (o *FactoryWorkOrder) IsClosed() bool {
	return o.State == FactoryWorkOrderStateClosed
}

// IsDispatchable reports whether the work order can be dispatched to a line.
// Both `draft` and `open` work orders accept new dispatches; the first
// dispatch from `draft` also transitions the order into `open` (see
// TransitionOnDispatch).
func (o *FactoryWorkOrder) IsDispatchable() bool {
	return o.State == FactoryWorkOrderStateDraft || o.State == FactoryWorkOrderStateOpen
}

type FactoryWorkOrderAssignee struct {
	WorkOrderID uuid.UUID `gorm:"primaryKey"`
	UserID      uuid.UUID `gorm:"primaryKey"`
	CreatedAt   time.Time

	User *User `gorm:"foreignKey:UserID"`
}

func (FactoryWorkOrderAssignee) TableName() string {
	return "factory_work_order_assignees"
}

// FactoryWorkOrderStatusUpdate captures a requested transition through the
// shared status FSM. Actor / run are optional and populate the emitted event.
type FactoryWorkOrderStatusUpdate struct {
	ToState  string
	Result   string
	Actor    *uuid.UUID
	Run      *factory.RunRef
	SkipSame bool // when true, no-op transitions to the current state succeed silently
}

// NOTE: this is only OK to be used in the workers.
// For APIs, always use the factory.FindWorkOrder method.
func FindUnscopedWorkOrder(db *gorm.DB, id uuid.UUID) (*FactoryWorkOrder, error) {
	var order FactoryWorkOrder
	err := db.Where("id = ?", id).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (o *FactoryWorkOrder) UpdateAssignees(tx *gorm.DB, assigneeIDs []uuid.UUID, updatedBy uuid.UUID) error {
	previousAssignees := o.AssigneeIDs()

	if err := o.ReplaceAssignees(tx, assigneeIDs); err != nil {
		return err
	}

	now := time.Now()
	o.UpdatedAt = now
	if err := tx.Model(o).Update("updated_at", now).Error; err != nil {
		return err
	}

	assigned, unassigned := assigneeDiff(previousAssignees, assigneeIDs)
	return o.RecordAssigneesUpdated(tx, updatedBy, assigned, unassigned)
}

// UpdateStatus is the single writer for the work order lifecycle. It validates
// the requested transition, updates the row, and records the corresponding
// events (always `order.status.updated`; plus `order.opened` when the order
// first enters `open`, and `order.closed` when it enters `closed`).
func (o *FactoryWorkOrder) UpdateStatus(db *gorm.DB, update FactoryWorkOrderStatusUpdate) error {
	toState := update.ToState
	if !slices.Contains(factoryWorkOrderStates, toState) {
		return fmt.Errorf("%w: unknown state %q", ErrFactoryWorkOrderInvalidState, toState)
	}

	if o.State == toState {
		if update.SkipSame {
			return nil
		}

		return fmt.Errorf("%w: work order is already %s", ErrFactoryWorkOrderInvalidState, toState)
	}

	if !slices.Contains(factoryWorkOrderAllowedTransitions[o.State], toState) {
		return fmt.Errorf("%w: cannot move from %s to %s", ErrFactoryWorkOrderInvalidState, o.State, toState)
	}

	nextResult := ""
	if toState == FactoryWorkOrderStateClosed {
		if !slices.Contains(factoryWorkOrderCloseResults, update.Result) {
			return fmt.Errorf("%w: closing requires a valid result (completed, rejected, failed)", ErrFactoryWorkOrderInvalidState)
		}
		nextResult = update.Result
	}

	fromState := o.State
	fromResult := o.Result
	now := time.Now()

	return db.Transaction(func(tx *gorm.DB) error {
		o.State = toState
		o.Result = nextResult
		o.UpdatedAt = now
		o.StateUpdatedAt = now

		err := tx.
			Model(o).
			Updates(map[string]any{
				"state":            o.State,
				"result":           o.Result,
				"updated_at":       o.UpdatedAt,
				"state_updated_at": o.StateUpdatedAt,
			}).
			Error
		if err != nil {
			return err
		}

		if err := o.RecordStatusUpdated(tx, update.Actor, update.Run, fromState, toState, fromResult, nextResult); err != nil {
			return err
		}

		//
		// Preserve the pre-existing coarse events so old UI/timeline logic
		// keeps working:
		//
		//   * `order.opened` fires only on the initial promotion from
		//     `draft` (the "first opened" event, mirroring a dispatch).
		//     Reopens from `closed` are covered by the
		//     `order.status.updated` event alone so the timeline can
		//     distinguish "opened" from "reopened".
		//   * `order.closed` fires whenever the order becomes `closed`.
		//
		if toState == FactoryWorkOrderStateOpen && fromState == FactoryWorkOrderStateDraft {
			if err := o.RecordOpened(tx, update.Actor); err != nil {
				return err
			}
		}

		if toState == FactoryWorkOrderStateClosed {
			if err := o.RecordClosed(tx, update.Actor, nextResult); err != nil {
				return err
			}
		}

		return nil
	})
}

// Close is kept for backwards compatibility with existing handlers. New code
// should call UpdateStatus directly.
func (o *FactoryWorkOrder) Close(db *gorm.DB, result string, closedBy *uuid.UUID) (*FactoryWorkOrder, error) {
	if o.IsClosed() {
		return o, nil
	}

	err := o.UpdateStatus(db, FactoryWorkOrderStatusUpdate{
		ToState:  FactoryWorkOrderStateClosed,
		Result:   result,
		Actor:    closedBy,
		SkipSame: true,
	})
	if err != nil {
		return nil, err
	}

	return o, nil
}

// TransitionOnDispatch is called when the work order is being dispatched to a
// factory line. Draft work orders promote to open at that moment; open ones
// stay put. Any other state means the work order is not dispatchable.
func (o *FactoryWorkOrder) TransitionOnDispatch(tx *gorm.DB, actor *uuid.UUID) error {
	if o.State == FactoryWorkOrderStateOpen {
		return nil
	}

	if o.State != FactoryWorkOrderStateDraft {
		return fmt.Errorf("%w: work order must be draft or open to dispatch", ErrFactoryWorkOrderInvalidState)
	}

	return o.UpdateStatus(tx, FactoryWorkOrderStatusUpdate{
		ToState: FactoryWorkOrderStateOpen,
		Actor:   actor,
	})
}

func (o *FactoryWorkOrder) FindActiveExecution(tx *gorm.DB) (*FactoryWorkOrderExecution, error) {
	var execution FactoryWorkOrderExecution
	err := tx.
		Where("work_order_id = ?", o.ID).
		Where("status IN ?", []string{
			FactoryWorkOrderExecutionStatusPending,
			FactoryWorkOrderExecutionStatusRunning,
		}).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func (o *FactoryWorkOrder) ReplaceAssignees(tx *gorm.DB, assigneeIDs []uuid.UUID) error {
	if err := tx.Where("work_order_id = ?", o.ID).Delete(&FactoryWorkOrderAssignee{}).Error; err != nil {
		return err
	}

	if len(assigneeIDs) == 0 {
		return nil
	}

	now := time.Now()
	assignees := make([]FactoryWorkOrderAssignee, 0, len(assigneeIDs))
	for _, assigneeID := range assigneeIDs {
		assignees = append(assignees, FactoryWorkOrderAssignee{
			WorkOrderID: o.ID,
			UserID:      assigneeID,
			CreatedAt:   now,
		})
	}

	return tx.Create(&assignees).Error
}

func (o *FactoryWorkOrder) ListEvents(tx *gorm.DB, limit int, before *time.Time) ([]FactoryWorkOrderEvent, error) {
	query := tx.Where("work_order_id = ?", o.ID)

	if limit > 0 {
		query = query.Limit(limit)
	}

	if before != nil {
		query = query.Where("created_at < ?", before)
	}

	var events []FactoryWorkOrderEvent
	err := query.
		Order("created_at DESC").
		Order("id DESC").
		Find(&events).
		Error
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (o *FactoryWorkOrder) CountEvents(tx *gorm.DB) (int64, error) {
	var count int64

	err := tx.
		Model(&FactoryWorkOrderEvent{}).
		Where("work_order_id = ?", o.ID).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (o *FactoryWorkOrder) Ref() *factory.WorkOrderRef {
	return &factory.WorkOrderRef{
		ID:    o.ID,
		Title: o.Title,
	}
}

func (o *FactoryWorkOrder) RecordOpened(tx *gorm.DB, createdBy *uuid.UUID) error {
	data := factory.WorkOrderOpened{
		Order: o.Ref(),
	}
	if createdBy != nil {
		data.User = &factory.UserRef{ID: *createdBy}
	}

	return o.recordEvent(tx, factory.EventTypeOrderOpened, data)
}

func (o *FactoryWorkOrder) RecordClosed(tx *gorm.DB, closedBy *uuid.UUID, result string) error {
	data := factory.WorkOrderClosed{
		Order:  o.Ref(),
		Result: &result,
	}
	if closedBy != nil {
		data.User = &factory.UserRef{ID: *closedBy}
	}

	return o.recordEvent(tx, factory.EventTypeOrderClosed, data)
}

func (o *FactoryWorkOrder) RecordStatusUpdated(
	tx *gorm.DB,
	actor *uuid.UUID,
	run *factory.RunRef,
	fromState, toState, fromResult, toResult string,
) error {
	data := factory.WorkOrderStatusUpdated{
		Order:      o.Ref(),
		Run:        run,
		FromState:  fromState,
		ToState:    toState,
		FromResult: fromResult,
		ToResult:   toResult,
	}
	if actor != nil {
		data.User = &factory.UserRef{ID: *actor}
	}

	return o.recordEvent(tx, factory.EventTypeOrderStatusUpdated, data)
}

func (o *FactoryWorkOrder) RecordCommentAdded(
	tx *gorm.DB,
	body string,
	author factory.WorkOrderCommentAuthor,
	run *factory.RunRef,
) error {
	data := factory.WorkOrderCommentAdded{
		Order:  o.Ref(),
		Body:   body,
		Author: &author,
		Run:    run,
	}

	return o.recordEvent(tx, factory.EventTypeOrderCommentAdded, data)
}

func (o *FactoryWorkOrder) RecordArtifactAdded(
	tx *gorm.DB,
	artifact *factory.ArtifactRef,
	actor *uuid.UUID,
	run *factory.RunRef,
) error {
	data := factory.WorkOrderArtifactAdded{
		Order:    o.Ref(),
		Artifact: artifact,
		Run:      run,
	}
	if actor != nil {
		data.User = &factory.UserRef{ID: *actor}
	}

	return o.recordEvent(tx, factory.EventTypeOrderArtifactAdded, data)
}

func (o *FactoryWorkOrder) RecordAssigneesUpdated(
	tx *gorm.DB,
	updatedBy uuid.UUID,
	assigned []factory.UserRef,
	unassigned []factory.UserRef,
) error {
	if len(assigned) == 0 && len(unassigned) == 0 {
		return nil
	}

	data := factory.WorkOrderAssigneesUpdated{
		Order:      o.Ref(),
		User:       &factory.UserRef{ID: updatedBy},
		Assigned:   assigned,
		Unassigned: unassigned,
	}

	return o.recordEvent(tx, factory.EventTypeOrderAssigneesUpdated, data)
}

func (o *FactoryWorkOrder) AssigneeIDs() []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(o.Assignees))
	for _, assignee := range o.Assignees {
		ids = append(ids, assignee.UserID)
	}

	return ids
}

func (o *FactoryWorkOrder) recordEvent(tx *gorm.DB, eventType string, payload any) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	event := &FactoryWorkOrderEvent{
		ID:          uuid.New(),
		WorkOrderID: o.ID,
		Type:        eventType,
		Data:        datatypes.JSON(jsonData),
		CreatedAt:   time.Now(),
	}

	return tx.Create(event).Error
}

// IsValidWorkOrderCommentAuthorKind reports whether the given author kind is
// accepted by RecordCommentAdded / API handlers.
func IsValidWorkOrderCommentAuthorKind(kind string) bool {
	switch kind {
	case factory.CommentAuthorKindUser,
		factory.CommentAuthorKindAutomation,
		factory.CommentAuthorKindSystem:
		return true
	}
	return false
}

func assigneeDiff(previousIDs, nextIDs []uuid.UUID) (assigned, unassigned []factory.UserRef) {
	previous := make(map[uuid.UUID]struct{}, len(previousIDs))
	for _, id := range previousIDs {
		previous[id] = struct{}{}
	}

	next := make(map[uuid.UUID]struct{}, len(nextIDs))
	for _, id := range nextIDs {
		next[id] = struct{}{}
	}

	for id := range next {
		if _, ok := previous[id]; !ok {
			assigned = append(assigned, factory.UserRef{ID: id})
		}
	}

	for id := range previous {
		if _, ok := next[id]; !ok {
			unassigned = append(unassigned, factory.UserRef{ID: id})
		}
	}

	return assigned, unassigned
}
