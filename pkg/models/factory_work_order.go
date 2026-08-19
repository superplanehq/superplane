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
	"gorm.io/gorm/clause"
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

	// Allowed close results per source state. A draft was never dispatched,
	// so it can only be `rejected` (abandoned). Open orders may terminate
	// with any of the three results depending on outcome.
	factoryWorkOrderCloseResultsByFromState = map[string][]string{
		FactoryWorkOrderStateDraft: {FactoryWorkOrderResultRejected},
		FactoryWorkOrderStateOpen: {
			FactoryWorkOrderResultCompleted,
			FactoryWorkOrderResultRejected,
			FactoryWorkOrderResultFailed,
		},
	}

	// Allowed transitions. `open → draft` is "back to draft"; `closed →
	// open` is reopen; `draft → closed` is "abandon before dispatch"
	// (rejected only). See TransitionOnDispatch for the draft → open promotion.
	factoryWorkOrderAllowedTransitions = map[string][]string{
		FactoryWorkOrderStateDraft:  {FactoryWorkOrderStateOpen, FactoryWorkOrderStateClosed},
		FactoryWorkOrderStateOpen:   {FactoryWorkOrderStateClosed, FactoryWorkOrderStateDraft},
		FactoryWorkOrderStateClosed: {FactoryWorkOrderStateOpen},
	}
)

type FactoryWorkOrder struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	Number         int64
	Title          string
	Description    string
	State          string
	Result         string
	CreatedByID    *uuid.UUID
	SourceRunID    *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time

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

// IsDispatchable reports whether the work order can be dispatched;
// the first dispatch from `draft` also promotes it to `open`.
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
// shared status FSM. Actor / Automation / Run + App are optional attribution
// fields that populate the emitted `order.status.updated` event.
type FactoryWorkOrderStatusUpdate struct {
	ToState    string
	Result     string
	Actor      *uuid.UUID
	Automation *factory.AutomationRef
	Run        *factory.RunRef
	App        *factory.AppRef
	SkipSame   bool // when true, no-op transitions to the current state succeed silently
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

func ResolveFactoryWorkOrderCreatorAutomations(
	tx *gorm.DB,
	orders []FactoryWorkOrder,
) (map[uuid.UUID]*factory.AutomationRef, error) {
	orderIDsByRunID := map[uuid.UUID][]uuid.UUID{}
	runIDs := make([]uuid.UUID, 0, len(orders))
	for i := range orders {
		if orders[i].SourceRunID == nil {
			continue
		}

		runID := *orders[i].SourceRunID
		if _, exists := orderIDsByRunID[runID]; !exists {
			runIDs = append(runIDs, runID)
		}
		orderIDsByRunID[runID] = append(orderIDsByRunID[runID], orders[i].ID)
	}

	if len(runIDs) == 0 {
		return map[uuid.UUID]*factory.AutomationRef{}, nil
	}

	var runs []CanvasRun
	if err := tx.Where("id IN ?", runIDs).Find(&runs).Error; err != nil {
		return nil, err
	}

	canvasIDs, nodeKeys := collectCreatorAutomationKeys(runs)
	canvasesByID, err := loadCreatorAutomationCanvases(tx, canvasIDs)
	if err != nil {
		return nil, err
	}
	nodesByKey, err := loadCreatorAutomationNodes(tx, canvasIDs, nodeKeys)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]*factory.AutomationRef, len(orders))
	for i := range runs {
		run := &runs[i]
		ref := &factory.AutomationRef{
			AppID:  run.WorkflowID,
			NodeID: run.NodeID,
		}
		ref.AppName = canvasesByID[run.WorkflowID]
		ref.NodeName = nodesByKey[creatorAutomationNodeKey{CanvasID: run.WorkflowID, NodeID: run.NodeID}]

		for _, orderID := range orderIDsByRunID[run.ID] {
			result[orderID] = ref
		}
	}

	return result, nil
}

func (o *FactoryWorkOrder) UpdateAssignees(tx *gorm.DB, assigneeIDs []uuid.UUID, updatedBy uuid.UUID) error {
	previousAssignees := o.AssigneeIDs()

	if err := o.ReplaceAssignees(tx, assigneeIDs); err != nil {
		return err
	}

	// Omit associations: `o.Assignees` is stale after ReplaceAssignees, and
	// GORM would otherwise re-save it as part of this update, reverting the
	// change we just made.
	now := time.Now()
	o.UpdatedAt = now
	if err := tx.Model(o).Omit(clause.Associations).Update("updated_at", now).Error; err != nil {
		return err
	}

	// Keep the in-memory association in sync with what was just written.
	o.Assignees = make([]FactoryWorkOrderAssignee, 0, len(assigneeIDs))
	for _, assigneeID := range assigneeIDs {
		o.Assignees = append(o.Assignees, FactoryWorkOrderAssignee{WorkOrderID: o.ID, UserID: assigneeID})
	}

	assigned, unassigned := assigneeDiff(previousAssignees, assigneeIDs)
	return o.RecordAssigneesUpdated(tx, updatedBy, assigned, unassigned)
}

// UpdateStatus is the single writer for the work order lifecycle: it
// validates the transition, updates the row, and records an
// `order.status.updated` event enriched with the caller's attribution
// (actor / automation / run + app). On the initial `draft → open` we also
// snapshot the originating run/app from `o.SourceRunID`, so the timeline
// can always trace an order back to the run that created it.
//
// The bool return reports whether a transition was actually recorded:
// `true` on a real state change, `false` when SkipSame swallowed a no-op
// (target state equals current state). Callers that fan out an
// `order.status.updated` event downstream must check this so a re-run
// doesn't emit a phantom transition — see FactoryContext.
func (o *FactoryWorkOrder) UpdateStatus(db *gorm.DB, update FactoryWorkOrderStatusUpdate) (bool, error) {
	toState := update.ToState
	if !slices.Contains(factoryWorkOrderStates, toState) {
		return false, fmt.Errorf("%w: unknown state %q", ErrFactoryWorkOrderInvalidState, toState)
	}

	if o.State == toState {
		if update.SkipSame {
			return false, nil
		}

		return false, fmt.Errorf("%w: work order is already %s", ErrFactoryWorkOrderInvalidState, toState)
	}

	fromState := o.State
	if !slices.Contains(factoryWorkOrderAllowedTransitions[fromState], toState) {
		return false, fmt.Errorf("%w: cannot move from %s to %s", ErrFactoryWorkOrderInvalidState, fromState, toState)
	}

	nextResult := ""
	if toState == FactoryWorkOrderStateClosed {
		allowed := factoryWorkOrderCloseResultsByFromState[fromState]
		if !slices.Contains(allowed, update.Result) {
			return false, fmt.Errorf(
				"%w: closing from %s requires result in %v (got %q)",
				ErrFactoryWorkOrderInvalidState, fromState, allowed, update.Result,
			)
		}
		nextResult = update.Result
	}

	fromResult := o.Result
	now := time.Now()

	err := db.Transaction(func(tx *gorm.DB) error {
		// Reverting `open → draft` while a line dispatch is still active would
		// desync the FSM from the executor. Mirror the dispatch guard here.
		if fromState == FactoryWorkOrderStateOpen && toState == FactoryWorkOrderStateDraft {
			if err := o.ensureNoActiveLineDispatch(tx); err != nil {
				return err
			}
		}

		o.State = toState
		o.Result = nextResult
		o.UpdatedAt = now

		err := tx.
			Model(o).
			Updates(map[string]any{
				"state":      o.State,
				"result":     o.Result,
				"updated_at": o.UpdatedAt,
			}).
			Error
		if err != nil {
			return err
		}

		// A closing order abandons any traversal still waiting in a step's
		// queue; a queued dispatch has no run to finish it later.
		if toState == FactoryWorkOrderStateClosed {
			if err := o.dropQueuedLineWork(tx); err != nil {
				return err
			}
		}

		run, app := update.Run, update.App
		if fromState == FactoryWorkOrderStateDraft && toState == FactoryWorkOrderStateOpen && o.SourceRunID != nil {
			// Snapshot the originating canvas run so downstream consumers can
			// trace back to the trigger without needing to load the order.
			sourceRun, sourceApp, err := o.loadSourceRunRefs(tx)
			if err != nil {
				return err
			}
			// Explicit caller-provided refs win, otherwise use the snapshot.
			if run == nil {
				run = sourceRun
			}
			if app == nil {
				app = sourceApp
			}
		}

		return o.RecordStatusUpdated(tx, statusUpdatedRecord{
			Actor:      update.Actor,
			Automation: update.Automation,
			Run:        run,
			App:        app,
			FromState:  fromState,
			ToState:    toState,
			FromResult: fromResult,
			ToResult:   nextResult,
		})
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// loadSourceRunRefs resolves the originating canvas run + app for an order
// created from a canvas run. Only called when o.SourceRunID is non-nil.
func (o *FactoryWorkOrder) loadSourceRunRefs(tx *gorm.DB) (*factory.RunRef, *factory.AppRef, error) {
	run, err := FindUnscopedCanvasRun(tx, *o.SourceRunID)
	if err != nil {
		return nil, nil, err
	}

	runRef := &factory.RunRef{ID: run.ID, State: run.State}
	if run.Result != "" {
		result := run.Result
		runRef.Result = &result
	}
	return runRef, &factory.AppRef{ID: run.WorkflowID}, nil
}

// Close is kept for backwards compatibility with existing handlers. New code
// should call UpdateStatus directly.
func (o *FactoryWorkOrder) Close(db *gorm.DB, result string, closedBy *uuid.UUID) (*FactoryWorkOrder, error) {
	if o.IsClosed() {
		return o, nil
	}

	if _, err := o.UpdateStatus(db, FactoryWorkOrderStatusUpdate{
		ToState:  FactoryWorkOrderStateClosed,
		Result:   result,
		Actor:    closedBy,
		SkipSame: true,
	}); err != nil {
		return nil, err
	}

	return o, nil
}

// TransitionOnDispatch promotes a draft order to open; open is a no-op.
// Any other state rejects the dispatch.
func (o *FactoryWorkOrder) TransitionOnDispatch(tx *gorm.DB, actor *uuid.UUID) error {
	if o.State == FactoryWorkOrderStateOpen {
		return nil
	}

	if o.State != FactoryWorkOrderStateDraft {
		return fmt.Errorf("%w: work order must be draft or open to dispatch", ErrFactoryWorkOrderInvalidState)
	}

	_, err := o.UpdateStatus(tx, FactoryWorkOrderStatusUpdate{
		ToState: FactoryWorkOrderStateOpen,
		Actor:   actor,
	})
	return err
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

// ListComments returns the work order's comment thread (`order.comment.added`
// events), oldest first — unlike ListEvents (which is DESC for activity
// feeds), a comment thread reads chronologically oldest→newest.
func (o *FactoryWorkOrder) ListComments(tx *gorm.DB) ([]FactoryWorkOrderEvent, error) {
	var events []FactoryWorkOrderEvent
	err := tx.
		Where("work_order_id = ?", o.ID).
		Where("type = ?", factory.EventTypeOrderCommentAdded).
		Order("created_at ASC").
		Order("id ASC").
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

// statusUpdatedRecord bundles the attribution + transition fields for
// `order.status.updated`. Keeps RecordStatusUpdated's signature small so
// callers (UpdateStatus, CreateWorkOrder) read cleanly at the call site.
type statusUpdatedRecord struct {
	Actor      *uuid.UUID
	Automation *factory.AutomationRef
	Run        *factory.RunRef
	App        *factory.AppRef
	FromState  string
	ToState    string
	FromResult string
	ToResult   string
}

func (o *FactoryWorkOrder) RecordStatusUpdated(tx *gorm.DB, r statusUpdatedRecord) error {
	data := factory.WorkOrderStatusUpdated{
		Order:      o.Ref(),
		Automation: r.Automation,
		Run:        r.Run,
		App:        r.App,
		FromState:  r.FromState,
		ToState:    r.ToState,
		FromResult: r.FromResult,
		ToResult:   r.ToResult,
	}
	if r.Actor != nil {
		data.User = &factory.UserRef{ID: *r.Actor}
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
	automation *factory.AutomationRef,
	run *factory.RunRef,
) error {
	data := factory.WorkOrderArtifactAdded{
		Order:      o.Ref(),
		Artifact:   artifact,
		Automation: automation,
		Run:        run,
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
		factory.CommentAuthorKindAutomation:
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

type creatorAutomationNodeKey struct {
	CanvasID uuid.UUID
	NodeID   string
}

func collectCreatorAutomationKeys(runs []CanvasRun) ([]uuid.UUID, []creatorAutomationNodeKey) {
	canvasSet := map[uuid.UUID]struct{}{}
	nodeSet := map[creatorAutomationNodeKey]struct{}{}
	for i := range runs {
		canvasSet[runs[i].WorkflowID] = struct{}{}
		if runs[i].NodeID != "" {
			nodeSet[creatorAutomationNodeKey{CanvasID: runs[i].WorkflowID, NodeID: runs[i].NodeID}] = struct{}{}
		}
	}

	canvasIDs := make([]uuid.UUID, 0, len(canvasSet))
	for id := range canvasSet {
		canvasIDs = append(canvasIDs, id)
	}
	nodeKeys := make([]creatorAutomationNodeKey, 0, len(nodeSet))
	for key := range nodeSet {
		nodeKeys = append(nodeKeys, key)
	}
	return canvasIDs, nodeKeys
}

func loadCreatorAutomationCanvases(tx *gorm.DB, canvasIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	var canvases []Canvas
	if err := tx.Unscoped().Where("id IN ?", canvasIDs).Find(&canvases).Error; err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]string, len(canvases))
	for i := range canvases {
		result[canvases[i].ID] = canvases[i].Name
	}
	return result, nil
}

func loadCreatorAutomationNodes(
	tx *gorm.DB,
	canvasIDs []uuid.UUID,
	keys []creatorAutomationNodeKey,
) (map[creatorAutomationNodeKey]string, error) {
	if len(keys) == 0 {
		return map[creatorAutomationNodeKey]string{}, nil
	}

	nodeIDs := make([]string, 0, len(keys))
	for i := range keys {
		nodeIDs = append(nodeIDs, keys[i].NodeID)
	}

	var nodes []CanvasNode
	err := tx.
		Where("workflow_nodes.workflow_id IN ?", canvasIDs).
		Where("workflow_nodes.node_id IN ?", nodeIDs).
		Find(&nodes).
		Error
	if err != nil {
		return nil, err
	}

	result := make(map[creatorAutomationNodeKey]string, len(nodes))
	for i := range nodes {
		key := creatorAutomationNodeKey{CanvasID: nodes[i].WorkflowID, NodeID: nodes[i].NodeID}
		result[key] = nodes[i].Name
	}
	return result, nil
}
