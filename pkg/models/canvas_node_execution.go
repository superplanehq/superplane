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

	//
	// Reference to the previous execution.
	// This is what allows us to build execution chains,
	// from any execution.
	//
	PreviousExecutionID *uuid.UUID

	//
	// Reference to the parent execution.
	// This is used for node executions inside of a blueprint node,
	// to reference the parent blueprint node execution.
	//
	ParentExecutionID *uuid.UUID

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

func LockCanvasNodeExecution(tx *gorm.DB, id uuid.UUID) (*CanvasNodeExecution, error) {
	var execution CanvasNodeExecution

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", id).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func CreatePendingChildExecution(tx *gorm.DB, parent *CanvasNodeExecution, childNodeID string, config map[string]any) (*CanvasNodeExecution, error) {
	now := time.Now()
	execution := CanvasNodeExecution{
		WorkflowID:          parent.WorkflowID,
		RootEventID:         parent.RootEventID,
		EventID:             parent.EventID,
		PreviousExecutionID: &parent.ID,
		ParentExecutionID:   &parent.ID,
		NodeID:              fmt.Sprintf("%s:%s", parent.NodeID, childNodeID),
		State:               CanvasNodeExecutionStatePending,
		Configuration:       datatypes.NewJSONType(config),
		CreatedAt:           &now,
		UpdatedAt:           &now,
	}

	err := tx.Create(&execution).Error
	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func ListPendingNodeExecutions() ([]CanvasNodeExecution, error) {
	var executions []CanvasNodeExecution
	query := database.Conn().
		Where("state = ?", CanvasNodeExecutionStatePending).
		Order("created_at DESC")

	err := query.Find(&executions).Error
	if err != nil {
		return nil, err
	}

	return executions, nil
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

func ListNodeExecutionsForRootEvents(rootEventIDs []uuid.UUID) ([]CanvasNodeExecution, error) {
	if len(rootEventIDs) == 0 {
		return []CanvasNodeExecution{}, nil
	}

	var executions []CanvasNodeExecution
	err := database.Conn().
		Where("root_event_id IN ?", rootEventIDs).
		Order("created_at ASC").
		Find(&executions).
		Error
	if err != nil {
		return nil, err
	}

	return executions, nil
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

func FindChildExecutionsForMultiple(parentExecutionIDs []string) ([]CanvasNodeExecution, error) {
	var executions []CanvasNodeExecution
	err := database.Conn().
		Where("parent_execution_id IN ?", parentExecutionIDs).
		Find(&executions).
		Error

	if err != nil {
		return nil, err
	}

	return executions, nil
}

func FindChildExecutions(parentExecutionID uuid.UUID, states []string) ([]CanvasNodeExecution, error) {
	return FindChildExecutionsInTransaction(database.Conn(), parentExecutionID, states)
}

func FindChildExecutionsInTransaction(tx *gorm.DB, parentExecutionID uuid.UUID, states []string) ([]CanvasNodeExecution, error) {
	var executions []CanvasNodeExecution
	err := tx.
		Where("parent_execution_id = ?", parentExecutionID).
		Where("state IN ?", states).
		Find(&executions).
		Error

	if err != nil {
		return nil, err
	}

	return executions, nil
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

func (e *CanvasNodeExecution) GetParentExecutionID() string {
	if e.ParentExecutionID == nil {
		return ""
	}

	return e.ParentExecutionID.String()
}

func (e *CanvasNodeExecution) Start() error {
	return e.StartInTransaction(database.Conn())
}

func (e *CanvasNodeExecution) StartInTransaction(tx *gorm.DB) error {
	// Just a sanity check that we are not trying to start and already started execution.
	if e.State != CanvasNodeExecutionStatePending {
		return fmt.Errorf("cannot start execution %s in state %s", e.ID, e.State)
	}

	//
	// Update the execution state to started.
	//
	return tx.Model(e).
		Update("state", CanvasNodeExecutionStateStarted).
		Update("updated_at", time.Now()).
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
				Data:        datatypes.NewJSONType(event),
				ExecutionID: &e.ID,
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
		if node.State != CanvasNodeStatePaused {
			err = node.UpdateState(tx, CanvasNodeStateReady)
			if err != nil {
				return nil, err
			}
		}
	}

	//
	// Update execution state
	//
	err = tx.Model(e).
		Updates(map[string]interface{}{
			"state":      CanvasNodeExecutionStateFinished,
			"result":     CanvasNodeExecutionResultPassed,
			"updated_at": &now,
		}).Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func (e *CanvasNodeExecution) Fail(reason, message string) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		return e.FailInTransaction(tx, reason, message)
	})
}

func (e *CanvasNodeExecution) FailInTransaction(tx *gorm.DB, reason, message string) error {
	now := time.Now()

	err := tx.Model(e).
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
		if node.State != CanvasNodeStatePaused {
			err := node.UpdateState(tx, CanvasNodeStateReady)
			if err != nil {
				return err
			}
		}
	}

	//
	// Since an execution failure does not emit anything,
	// we need to update the parent execution here too,
	// if this execution is a child one.
	//
	if e.ParentExecutionID != nil {
		parent, err := FindNodeExecution(e.WorkflowID, *e.ParentExecutionID)
		if err != nil {
			return err
		}

		return parent.FailInTransaction(tx, reason, message)
	}

	return nil
}

func (e *CanvasNodeExecution) Cancel(cancelledBy *uuid.UUID) error {
	return e.CancelInTransaction(database.Conn(), cancelledBy)
}

func (e *CanvasNodeExecution) CancelInTransaction(tx *gorm.DB, cancelledBy *uuid.UUID) error {
	now := time.Now()

	err := tx.Model(e).
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
		if node.State != CanvasNodeStatePaused {
			err := node.UpdateState(tx, CanvasNodeStateReady)
			if err != nil {
				return err
			}
		}
	}

	return nil
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

// FindLastExecutionPerNode finds the most recent execution for each node in a workflow
// using DISTINCT ON to get one execution per node_id, ordered by created_at DESC
// Only returns executions for nodes that have not been deleted
func FindLastExecutionPerNode(workflowID uuid.UUID) ([]CanvasNodeExecution, error) {
	var executions []CanvasNodeExecution
	err := database.Conn().
		Raw(`
			SELECT DISTINCT ON (wne.node_id) wne.*
			FROM workflow_node_executions wne
			INNER JOIN workflow_nodes wn
				ON wne.workflow_id = wn.workflow_id
				AND wne.node_id = wn.node_id
			WHERE wne.workflow_id = ?
			AND wne.parent_execution_id IS NULL
			AND wn.deleted_at IS NULL
			ORDER BY wne.node_id, wne.created_at DESC
		`, workflowID).
		Scan(&executions).
		Error

	if err != nil {
		return nil, err
	}

	return executions, nil
}
