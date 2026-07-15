package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CanvasNodeExecutionStatePending  = "pending"
	CanvasNodeExecutionStateStarted  = "started"
	CanvasNodeExecutionStateFinished = "finished"

	CanvasNodeExecutionResultPassed    = "passed"
	CanvasNodeExecutionResultFailed    = "failed"
	CanvasNodeExecutionResultCancelled = "cancelled"

	CanvasNodeExecutionResultReasonOk            = "ok"
	CanvasNodeExecutionResultReasonError         = "error"
	CanvasNodeExecutionResultReasonErrorResolved = "error_resolved"
)

type CanvasNodeExecution struct {
	ID         uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	WorkflowID uuid.UUID
	NodeID     string
	CreatedAt  *time.Time
	UpdatedAt  *time.Time

	//
	// Reference to the root WorkflowEvent record that started
	// this whole execution chain.
	//
	// This gives us an easy way to find all the executions
	// for that event with a simple query.
	//
	RootEventID uuid.UUID
	RunID       uuid.UUID

	//
	// Reference to the previous execution.
	// This is what allows us to build execution chains,
	// from any execution.
	//
	PreviousExecutionID *uuid.UUID

	//
	// The reference to a WorkflowEvent record,
	// which holds the input for this execution.
	//
	EventID uuid.UUID

	//
	// State management fields.
	//
	State         string
	Result        string
	ResultReason  string
	ResultMessage string
	CancelledBy   *uuid.UUID

	//
	// Components can store metadata about each execution here.
	// This allows them to control the behavior of each execution.
	//
	Metadata datatypes.JSONType[map[string]any]

	//
	// The configuration is copied from the node.
	// This enables us to allow node configuration updates
	// while executions are running.
	// Only new executions will use the new node configuration.
	//
	Configuration datatypes.JSONType[map[string]any]
}

func (e *CanvasNodeExecution) TableName() string {
	return "workflow_node_executions"
}

func (e *CanvasNodeExecution) BeforeCreate(tx *gorm.DB) error {
	if e.RunID != uuid.Nil {
		return nil
	}

	run, err := FindCanvasRunByRootEventInTransaction(tx, e.RootEventID)
	if err != nil {
		return err
	}

	e.RunID = run.ID
	return nil
}

func ListPendingNodeExecutions() ([]CanvasNodeExecution, error) {
	var executions []CanvasNodeExecution
	query := database.Conn().
		Table("workflow_node_executions").
		Select("workflow_node_executions.*").
		Where("workflow_node_executions.state = ?", CanvasNodeExecutionStatePending).
		Order("workflow_node_executions.created_at DESC")

	err := withActiveCanvas(query, "workflow_node_executions.workflow_id").
		Find(&executions).
		Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func LockPendingNodeExecutionInActiveCanvas(tx *gorm.DB, id uuid.UUID) (*CanvasNodeExecution, error) {
	var execution CanvasNodeExecution

	query := tx.
		Table("workflow_node_executions").
		Select("workflow_node_executions.*").
		Clauses(clause.Locking{
			Strength: lockingForUpdateNoKey,
			Table:    clause.Table{Name: "workflow_node_executions"},
			Options:  "SKIP LOCKED",
		}).
		Where("workflow_node_executions.id = ?", id).
		Where("workflow_node_executions.state = ?", CanvasNodeExecutionStatePending)

	err := withActiveCanvas(query, "workflow_node_executions.workflow_id").
		First(&execution).
		Error
	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func ListNodeExecutions(workflowID uuid.UUID, nodeID string, states []string, results []string, limit int, beforeTime *time.Time) ([]CanvasNodeExecution, error) {
	var executions []CanvasNodeExecution
	query := database.Conn().
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Order("created_at DESC").
		Limit(int(limit))

	if len(states) > 0 {
		query = query.Where("state IN ?", states)
	}

	if len(results) > 0 {
		query = query.Where("result IN ?", results)
	}

	if beforeTime != nil {
		query = query.Where("created_at < ?", beforeTime)
	}

	err := query.Find(&executions).Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func ListActiveNodeExecutions(tx *gorm.DB, workflowID uuid.UUID, nodeID string) ([]CanvasNodeExecution, error) {
	var executions []CanvasNodeExecution
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Where("state IN ?", []string{CanvasNodeExecutionStatePending, CanvasNodeExecutionStateStarted}).
		Find(&executions).
		Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func ListParentExecutionsForRootEvents(canvasID uuid.UUID, rootEventIDs []uuid.UUID) ([]CanvasNodeExecution, error) {
	return ListParentExecutionsForRootEventsInTransaction(database.Conn(), canvasID, rootEventIDs)
}

func ListParentExecutionsForRootEventsInTransaction(tx *gorm.DB, canvasID uuid.UUID, rootEventIDs []uuid.UUID) ([]CanvasNodeExecution, error) {
	if len(rootEventIDs) == 0 {
		return []CanvasNodeExecution{}, nil
	}

	var executions []CanvasNodeExecution
	query := tx.
		Where("workflow_id = ?", canvasID).
		Where("root_event_id IN ?", rootEventIDs).
		Order("created_at ASC")

	err := query.Find(&executions).Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func ListAllExecutionsForRootEvent(rootEventID uuid.UUID) ([]CanvasNodeExecution, error) {
	return ListAllExecutionsForRootEventInTransaction(database.Conn(), rootEventID)
}

func ListAllExecutionsForRootEventInTransaction(tx *gorm.DB, rootEventID uuid.UUID) ([]CanvasNodeExecution, error) {
	var executions []CanvasNodeExecution
	query := tx.
		Where("root_event_id = ?", rootEventID).
		Order("created_at ASC")

	err := query.Find(&executions).Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func CountActiveNodeExecutionsForRootEventInTransaction(tx *gorm.DB, rootEventID uuid.UUID) (int64, error) {
	var count int64

	err := tx.
		Model(&CanvasNodeExecution{}).
		Where("root_event_id = ?", rootEventID).
		Where("state IN ?", []string{CanvasNodeExecutionStatePending, CanvasNodeExecutionStateStarted}).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func CountNodeExecutions(workflowID uuid.UUID, nodeID string, states []string, results []string) (int64, error) {
	var totalCount int64
	countQuery := database.Conn().
		Model(&CanvasNodeExecution{}).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID)

	if len(states) > 0 {
		countQuery = countQuery.Where("state IN ?", states)
	}

	if len(results) > 0 {
		countQuery = countQuery.Where("result IN ?", results)
	}

	if err := countQuery.Count(&totalCount).Error; err != nil {
		return 0, err
	}

	return totalCount, nil
}

func CountRunningExecutionsForNode(workflowID uuid.UUID, nodeID string) (int64, error) {
	return CountRunningExecutionsForNodeInTransaction(database.Conn(), workflowID, nodeID)
}

func CountRunningExecutionsForNodeInTransaction(tx *gorm.DB, workflowID uuid.UUID, nodeID string) (int64, error) {
	var runningCount int64
	err := tx.
		Model(&CanvasNodeExecution{}).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Where("state = ?", CanvasNodeExecutionStateStarted).
		Count(&runningCount).
		Error
	if err != nil {
		return 0, err
	}

	return runningCount, nil
}

func FindNodeExecution(workflowID, id uuid.UUID) (*CanvasNodeExecution, error) {
	return FindNodeExecutionInTransaction(database.Conn(), workflowID, id)
}

func FindNodeExecutionInTransaction(tx *gorm.DB, workflowID, id uuid.UUID) (*CanvasNodeExecution, error) {
	var execution CanvasNodeExecution
	err := tx.
		Where("id = ?", id).
		Where("workflow_id = ?", workflowID).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func FindNodeExecutionWithNodeID(workflowID, id uuid.UUID, nodeID string) (*CanvasNodeExecution, error) {
	return FindNodeExecutionWithNodeIDInTransaction(database.Conn(), workflowID, id, nodeID)
}

func FindNodeExecutionWithNodeIDInTransaction(tx *gorm.DB, workflowID, id uuid.UUID, nodeID string) (*CanvasNodeExecution, error) {
	var execution CanvasNodeExecution
	err := tx.
		Where("id = ?", id).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func FindNodeExecutionsByIDsInTransaction(tx *gorm.DB, workflowID uuid.UUID, executionIDs []uuid.UUID) ([]CanvasNodeExecution, error) {
	var executions []CanvasNodeExecution
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("id IN ?", executionIDs).
		Find(&executions).
		Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func FindNodeExecutionsByIDs(workflowID uuid.UUID, executionIDs []uuid.UUID) ([]CanvasNodeExecution, error) {
	return FindNodeExecutionsByIDsInTransaction(database.Conn(), workflowID, executionIDs)
}

func ResolveExecutionErrorsInTransaction(tx *gorm.DB, workflowID uuid.UUID, executionIDs []uuid.UUID) error {
	now := time.Now()
	return tx.Model(&CanvasNodeExecution{}).
		Where("workflow_id = ?", workflowID).
		Where("id IN ?", executionIDs).
		Updates(map[string]any{
			"result_reason": CanvasNodeExecutionResultReasonErrorResolved,
			"updated_at":    &now,
		}).
		Error
}

func ResolveExecutionErrors(workflowID uuid.UUID, executionIDs []uuid.UUID) error {
	return ResolveExecutionErrorsInTransaction(database.Conn(), workflowID, executionIDs)
}

func (e *CanvasNodeExecution) GetPreviousExecutionID() string {
	if e.PreviousExecutionID == nil {
		return ""
	}

	return e.PreviousExecutionID.String()
}

func (e *CanvasNodeExecution) Start() error {
	return e.StartInTransaction(database.Conn())
}

func (e *CanvasNodeExecution) StartInTransaction(tx *gorm.DB) error {
	// Just a sanity check that we are not trying to start and already started execution.
	if e.State != CanvasNodeExecutionStatePending {
		return fmt.Errorf("cannot start execution %s in state %s", e.ID, e.State)
	}

	now := time.Now()
	e.State = CanvasNodeExecutionStateStarted
	e.UpdatedAt = &now

	return tx.Model(e).
		Update("state", CanvasNodeExecutionStateStarted).
		Update("updated_at", now).
		Error
}

func (e *CanvasNodeExecution) Pass(outputs map[string][]any) ([]CanvasEvent, error) {
	var events []CanvasEvent
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		events, err = e.PassInTransaction(tx, outputs)
		if err != nil {
			return err
		}
		return nil
	})

	return events, err
}

func (e *CanvasNodeExecution) PassInTransaction(tx *gorm.DB, channelOutputs map[string][]any) ([]CanvasEvent, error) {
	finished, err := e.IsFinishedInTransaction(tx)
	if err != nil {
		return nil, err
	}

	if finished {
		return []CanvasEvent{}, nil
	}

	now := time.Now()

	//
	// Create events for outputs
	//
	events := []CanvasEvent{}
	for channel, outputs := range channelOutputs {
		for _, event := range outputs {
			events = append(events, CanvasEvent{
				WorkflowID:  e.WorkflowID,
				NodeID:      e.NodeID,
				Channel:     channel,
				Data:        NewJSONValue(event),
				ExecutionID: &e.ID,
				RunID:       e.RunID,
				State:       CanvasEventStatePending,
				CreatedAt:   &now,
			})
		}
	}

	if len(events) > 0 {
		err := tx.Create(&events).Error
		if err != nil {
			return nil, fmt.Errorf("failed to create events: %w", err)
		}
	}

	//
	// Update the workflow node state to ready.
	//
	node, err := FindCanvasNode(tx, e.WorkflowID, e.NodeID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if node != nil {
		err = node.UpdateState(tx, CanvasNodeStateReady)
		if err != nil {
			return nil, err
		}
	}

	//
	// Update execution state
	//
	e.State = CanvasNodeExecutionStateFinished
	e.Result = CanvasNodeExecutionResultPassed
	e.UpdatedAt = &now

	err = tx.Model(e).
		Updates(map[string]interface{}{
			"state":      CanvasNodeExecutionStateFinished,
			"result":     CanvasNodeExecutionResultPassed,
			"updated_at": &now,
		}).Error

	if err != nil {
		return nil, err
	}

	if err := CompletePendingRequestsForExecutionInTransaction(tx, e.ID); err != nil {
		return nil, err
	}

	//
	// If execution produced events, we know for sure that the run is not finished yet.
	// If the events produced are terminal, the EventRouter will handle the run finalization.
	//
	if len(events) > 0 {
		return events, nil
	}

	return events, nil
}

func (e *CanvasNodeExecution) EmitOutputsInTransaction(tx *gorm.DB, channelOutputs map[string][]any) ([]CanvasEvent, error) {
	now := time.Now()

	events := []CanvasEvent{}
	for channel, outputs := range channelOutputs {
		for _, event := range outputs {
			events = append(events, CanvasEvent{
				WorkflowID:  e.WorkflowID,
				NodeID:      e.NodeID,
				Channel:     channel,
				Data:        NewJSONValue(event),
				ExecutionID: &e.ID,
				RunID:       e.RunID,
				State:       CanvasEventStatePending,
				CreatedAt:   &now,
			})
		}
	}

	if len(events) > 0 {
		err := tx.Create(&events).Error
		if err != nil {
			return nil, fmt.Errorf("failed to create events: %w", err)
		}
	}

	node, err := FindCanvasNode(tx, e.WorkflowID, e.NodeID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if node != nil {
		err = node.UpdateState(tx, CanvasNodeStateReady)
		if err != nil {
			return nil, err
		}
	}

	if e.State == CanvasNodeExecutionStatePending {
		if err := e.StartInTransaction(tx); err != nil {
			return nil, err
		}
	}

	return events, nil
}

func (e *CanvasNodeExecution) Fail(reason, message string) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		return e.FailInTransaction(tx, reason, message)
	})
}

func (e *CanvasNodeExecution) FailInTransaction(tx *gorm.DB, reason, message string) error {
	finished, err := e.IsFinishedInTransaction(tx)
	if err != nil {
		return err
	}

	if finished {
		return nil
	}

	now := time.Now()

	e.State = CanvasNodeExecutionStateFinished
	e.Result = CanvasNodeExecutionResultFailed
	e.ResultReason = reason
	e.ResultMessage = message
	e.UpdatedAt = &now

	err = tx.Model(e).
		Updates(map[string]interface{}{
			"state":          CanvasNodeExecutionStateFinished,
			"result":         CanvasNodeExecutionResultFailed,
			"result_reason":  reason,
			"result_message": message,
			"updated_at":     &now,
		}).Error

	if err != nil {
		return err
	}

	//
	// Update the workflow node state to ready.
	//
	node, err := FindCanvasNode(tx, e.WorkflowID, e.NodeID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if node != nil {
		err := node.UpdateState(tx, CanvasNodeStateReady)
		if err != nil {
			return err
		}
	}

	return CompletePendingRequestsForExecutionInTransaction(tx, e.ID)
}

func (e *CanvasNodeExecution) Cancel(cancelledBy *uuid.UUID) error {
	return e.CancelInTransaction(database.Conn(), cancelledBy)
}

func (e *CanvasNodeExecution) CancelInTransaction(tx *gorm.DB, cancelledBy *uuid.UUID) error {
	finished, err := e.IsFinishedInTransaction(tx)
	if err != nil {
		return err
	}

	if finished {
		return nil
	}

	now := time.Now()

	e.State = CanvasNodeExecutionStateFinished
	e.Result = CanvasNodeExecutionResultCancelled
	e.CancelledBy = cancelledBy
	e.UpdatedAt = &now

	err = tx.Model(e).
		Updates(map[string]interface{}{
			"state":        CanvasNodeExecutionStateFinished,
			"result":       CanvasNodeExecutionResultCancelled,
			"cancelled_by": cancelledBy,
			"updated_at":   &now,
		}).Error

	if err != nil {
		return err
	}

	node, err := FindCanvasNode(tx, e.WorkflowID, e.NodeID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if node != nil {
		err := node.UpdateState(tx, CanvasNodeStateReady)
		if err != nil {
			return err
		}
	}

	return CompletePendingRequestsForExecutionInTransaction(tx, e.ID)
}

func (e *CanvasNodeExecution) GetInput(tx *gorm.DB) (any, error) {
	event, err := FindCanvasEventInTransaction(tx, e.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to find initial event %s: %w", e.RootEventID, err)
	}

	return event.Data.Data(), nil
}

func (e *CanvasNodeExecution) GetOutputs() ([]CanvasEvent, error) {
	var events []CanvasEvent
	err := database.Conn().
		Where("execution_id = ?", e.ID).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func (e *CanvasNodeExecution) GetOutputsInTransaction(tx *gorm.DB) ([]CanvasEvent, error) {
	var events []CanvasEvent
	err := tx.
		Where("execution_id = ?", e.ID).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func (e *CanvasNodeExecution) IsFinishedInTransaction(tx *gorm.DB) (bool, error) {
	var execution CanvasNodeExecution
	err := tx.
		Select("state").
		Where("id = ?", e.ID).
		First(&execution).
		Error
	if err != nil {
		return false, err
	}

	return execution.State == CanvasNodeExecutionStateFinished, nil
}

func ListCanvasEventsForExecutionsInTransaction(tx *gorm.DB, executionIDs []uuid.UUID) ([]CanvasEvent, error) {
	var events []CanvasEvent
	err := tx.
		Where("execution_id IN ?", executionIDs).
		Find(&events).
		Error
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (e *CanvasNodeExecution) CreateRequest(tx *gorm.DB, reqType string, spec NodeExecutionRequestSpec, runAt *time.Time) error {
	return tx.Create(&CanvasNodeRequest{
		WorkflowID:  e.WorkflowID,
		NodeID:      e.NodeID,
		ExecutionID: &e.ID,
		ID:          uuid.New(),
		State:       NodeExecutionRequestStatePending,
		Type:        reqType,
		Spec:        datatypes.NewJSONType(spec),
		RunAt:       *runAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}).Error
}

// FindLastExecutionPerNode finds the most recent execution for each node in a workflow.
// Only returns executions for nodes that have not been deleted.
func FindLastExecutionPerNode(tx *gorm.DB, workflowID uuid.UUID) ([]CanvasNodeExecution, error) {
	var executions []CanvasNodeExecution
	err := tx.
		Raw(`
			SELECT wne.*
			FROM workflow_nodes wn
			INNER JOIN LATERAL (
				SELECT *
				FROM workflow_node_executions
				WHERE workflow_id = wn.workflow_id
				  AND node_id = wn.node_id
				ORDER BY created_at DESC
				LIMIT 1
			) wne ON true
			WHERE wn.workflow_id = ?
			  AND wn.deleted_at IS NULL
		`, workflowID).
		Scan(&executions).
		Error

	if err != nil {
		return nil, err
	}

	return executions, nil
}
