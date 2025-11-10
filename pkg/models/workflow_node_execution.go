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
	WorkflowNodeExecutionStatePending  = "pending"
	WorkflowNodeExecutionStateStarted  = "started"
	WorkflowNodeExecutionStateFinished = "finished"

	WorkflowNodeExecutionResultPassed = "passed"
	WorkflowNodeExecutionResultFailed = "failed"

	WorkflowNodeExecutionResultReasonError = "error"
)

type WorkflowNodeExecution struct {
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

func LockWorkflowNodeExecution(tx *gorm.DB, id uuid.UUID) (*WorkflowNodeExecution, error) {
	var execution WorkflowNodeExecution

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

func CreatePendingChildExecution(tx *gorm.DB, parent *WorkflowNodeExecution, childNodeID string, config map[string]any) (*WorkflowNodeExecution, error) {
	now := time.Now()
	execution := WorkflowNodeExecution{
		WorkflowID:          parent.WorkflowID,
		RootEventID:         parent.RootEventID,
		EventID:             parent.EventID,
		PreviousExecutionID: &parent.ID,
		ParentExecutionID:   &parent.ID,
		NodeID:              fmt.Sprintf("%s:%s", parent.NodeID, childNodeID),
		State:               WorkflowNodeExecutionStatePending,
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

func ListPendingNodeExecutions() ([]WorkflowNodeExecution, error) {
	var executions []WorkflowNodeExecution
	query := database.Conn().
		Where("state = ?", WorkflowNodeExecutionStatePending).
		Where("parent_execution_id IS NULL").
		Order("created_at DESC")

	err := query.Find(&executions).Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func ListNodeExecutions(workflowID uuid.UUID, nodeID string, states []string, results []string, limit int, beforeTime *time.Time) ([]WorkflowNodeExecution, error) {
	var executions []WorkflowNodeExecution
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

func CountNodeExecutions(workflowID uuid.UUID, nodeID string, states []string, results []string) (int64, error) {
	var totalCount int64
	countQuery := database.Conn().
		Model(&WorkflowNodeExecution{}).
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

func FindNodeExecution(workflowID, id uuid.UUID) (*WorkflowNodeExecution, error) {
	return FindNodeExecutionInTransaction(database.Conn(), workflowID, id)
}

func FindNodeExecutionInTransaction(tx *gorm.DB, workflowID, id uuid.UUID) (*WorkflowNodeExecution, error) {
	var execution WorkflowNodeExecution
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

func ListPendingChildExecutions() ([]WorkflowNodeExecution, error) {
	return ListPendingChildExecutionsInTransaction(database.Conn())
}

func ListPendingChildExecutionsInTransaction(tx *gorm.DB) ([]WorkflowNodeExecution, error) {
	var executions []WorkflowNodeExecution
	err := tx.
		Where("state = ?", WorkflowNodeExecutionStatePending).
		Where("parent_execution_id IS NOT NULL").
		Find(&executions).
		Error

	if err != nil {
		return nil, err
	}

	return executions, nil
}

func FindChildExecutionsForMultiple(parentExecutionIDs []string) ([]WorkflowNodeExecution, error) {
	var executions []WorkflowNodeExecution
	err := database.Conn().
		Where("parent_execution_id IN ?", parentExecutionIDs).
		Find(&executions).
		Error

	if err != nil {
		return nil, err
	}

	return executions, nil
}

func FindChildExecutions(parentExecutionID uuid.UUID, states []string) ([]WorkflowNodeExecution, error) {
	return FindChildExecutionsInTransaction(database.Conn(), parentExecutionID, states)
}

func FindChildExecutionsInTransaction(tx *gorm.DB, parentExecutionID uuid.UUID, states []string) ([]WorkflowNodeExecution, error) {
	var executions []WorkflowNodeExecution
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

func (e *WorkflowNodeExecution) GetPreviousExecutionID() string {
	if e.PreviousExecutionID == nil {
		return ""
	}

	return e.PreviousExecutionID.String()
}

func (e *WorkflowNodeExecution) GetParentExecutionID() string {
	if e.ParentExecutionID == nil {
		return ""
	}

	return e.ParentExecutionID.String()
}

func (e *WorkflowNodeExecution) Start() error {
	return e.StartInTransaction(database.Conn())
}

func (e *WorkflowNodeExecution) StartInTransaction(tx *gorm.DB) error {
	//
	// Update the execution state to started.
	//
	return tx.Model(e).
		Update("state", WorkflowNodeExecutionStateStarted).
		Update("updated_at", time.Now()).
		Error
}

func (e *WorkflowNodeExecution) Pass(outputs map[string][]any) ([]WorkflowEvent, error) {
	var events []WorkflowEvent
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

func (e *WorkflowNodeExecution) PassInTransaction(tx *gorm.DB, channelOutputs map[string][]any) ([]WorkflowEvent, error) {
	now := time.Now()

	//
	// Create events for outputs
	//
	events := []WorkflowEvent{}
	for channel, outputs := range channelOutputs {
		for _, event := range outputs {
			events = append(events, WorkflowEvent{
				WorkflowID:  e.WorkflowID,
				NodeID:      e.NodeID,
				Channel:     channel,
				Data:        datatypes.NewJSONType(event),
				ExecutionID: &e.ID,
				State:       WorkflowEventStatePending,
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
	node, err := FindWorkflowNode(tx, e.WorkflowID, e.NodeID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if node != nil {
		err = node.UpdateState(tx, WorkflowNodeStateReady)
		if err != nil {
			return nil, err
		}
	}

	//
	// Update execution state
	//
	err = tx.Model(e).
		Updates(map[string]interface{}{
			"state":      WorkflowNodeExecutionStateFinished,
			"result":     WorkflowNodeExecutionResultPassed,
			"updated_at": &now,
		}).Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func (e *WorkflowNodeExecution) Fail(reason, message string) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		return e.FailInTransaction(tx, reason, message)
	})
}

func (e *WorkflowNodeExecution) FailInTransaction(tx *gorm.DB, reason, message string) error {
	now := time.Now()

	err := tx.Model(e).
		Updates(map[string]interface{}{
			"state":          WorkflowNodeExecutionStateFinished,
			"result":         WorkflowNodeExecutionResultFailed,
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
	node, err := FindWorkflowNode(tx, e.WorkflowID, e.NodeID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if node != nil {
		err := node.UpdateState(tx, WorkflowNodeStateReady)
		if err != nil {
			return err
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

func (e *WorkflowNodeExecution) GetInput(tx *gorm.DB) (any, error) {
	event, err := FindWorkflowEventInTransaction(tx, e.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to find initial event %s: %w", e.RootEventID, err)
	}

	return event.Data.Data(), nil
}

func (e *WorkflowNodeExecution) GetOutputs() ([]WorkflowEvent, error) {
	var events []WorkflowEvent
	err := database.Conn().
		Where("execution_id = ?", e.ID).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func (e *WorkflowNodeExecution) GetOutputsInTransaction(tx *gorm.DB) ([]WorkflowEvent, error) {
	var events []WorkflowEvent
	err := tx.
		Where("execution_id = ?", e.ID).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func (e *WorkflowNodeExecution) CreateRequest(tx *gorm.DB, reqType string, spec NodeExecutionRequestSpec, runAt *time.Time) error {
	return tx.Create(&WorkflowNodeRequest{
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
func FindLastExecutionPerNode(workflowID uuid.UUID) ([]WorkflowNodeExecution, error) {
	var executions []WorkflowNodeExecution
	err := database.Conn().
		Raw(`
			SELECT DISTINCT ON (node_id) *
			FROM workflow_node_executions
			WHERE workflow_id = ?
			AND parent_execution_id IS NULL
			ORDER BY node_id, created_at DESC
		`, workflowID).
		Scan(&executions).
		Error

	if err != nil {
		return nil, err
	}

	return executions, nil
}
