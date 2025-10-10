package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	WorkflowNodeExecutionStatePending  = "pending"
	WorkflowNodeExecutionStateStarted  = "started"
	WorkflowNodeExecutionStateRouting  = "routing"
	WorkflowNodeExecutionStateFinished = "finished"

	WorkflowNodeExecutionResultPassed = "passed"
	WorkflowNodeExecutionResultFailed = "failed"

	WorkflowNodeExecutionResultReasonError = "error"
)

type WorkflowNodeExecution struct {
	ID         uuid.UUID
	WorkflowID uuid.UUID
	NodeID     string

	// Root event ID - shared by all executions triggered by the same initial event
	RootEventID uuid.UUID

	// Sequential flow - references to previous execution that provides inputs
	PreviousExecutionID  *uuid.UUID
	PreviousOutputBranch *string
	PreviousOutputIndex  *int

	// Blueprint hierarchy - reference to the blueprint node execution that spawned this
	ParentExecutionID *uuid.UUID

	// Blueprint context (if this execution is running inside a blueprint)
	BlueprintID *uuid.UUID

	// State machine
	State         string
	Result        string
	ResultReason  string
	ResultMessage string

	//
	// The outputs of the node execution.
	// Note that this is a map[string][]any type.
	// The key in the map is the output branch name.
	// A node can emit multiple events as part of the same output.
	// The subsequent node in the flow will unpack that and create
	// multiple child executions.
	//
	// Inputs are NOT stored here - they are derived from the parent execution.
	//
	Outputs datatypes.JSONType[map[string][]any]

	//
	// Components can store metadata about themselves here.
	// This allows them to control their behavior.
	//
	Metadata datatypes.JSONType[map[string]any]

	//
	// The configuration is copied from the node.
	// This enables us to allow node configuration updates
	// while executions are running.
	// Only new executions will use the new node configuration.
	//
	Configuration datatypes.JSONType[map[string]any]

	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func FindPendingNodeExecutions() ([]WorkflowNodeExecution, error) {
	var executions []WorkflowNodeExecution

	// Get the oldest pending execution for each (workflow_id, node_id) pair
	// This prevents processing multiple pending executions for the same node concurrently
	err := database.Conn().
		Raw(`
			SELECT DISTINCT ON (workflow_id, node_id) *
			FROM workflow_node_executions
			WHERE state = ?
			ORDER BY workflow_id, node_id, created_at ASC
		`, WorkflowNodeExecutionStatePending).
		Find(&executions).
		Error

	if err != nil {
		return nil, err
	}

	return executions, nil
}

func FindRoutingNodeExecutions() ([]WorkflowNodeExecution, error) {
	var executions []WorkflowNodeExecution
	err := database.Conn().
		Where("state = ?", WorkflowNodeExecutionStateRouting).
		Find(&executions).
		Error

	if err != nil {
		return nil, err
	}

	return executions, nil
}

func FindNodeExecution(id uuid.UUID) (*WorkflowNodeExecution, error) {
	var execution WorkflowNodeExecution
	err := database.Conn().
		Where("id = ?", id).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func FindLastNodeExecutionForNode(workflowID uuid.UUID, nodeID string, states []string) (*WorkflowNodeExecution, error) {
	var execution WorkflowNodeExecution
	err := database.Conn().
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Where("state IN ?", states).
		Order("updated_at DESC").
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func (e *WorkflowNodeExecution) Start() error {
	return e.StartInTransaction(database.Conn())
}

func (e *WorkflowNodeExecution) StartInTransaction(tx *gorm.DB) error {
	now := time.Now()
	return tx.Model(e).
		Updates(map[string]interface{}{
			"state":      WorkflowNodeExecutionStateStarted,
			"updated_at": &now,
		}).Error
}

//
// Passing and failing an execution just updates its result,
// and moves its state to routing, so the execution router can take it,
// and route it to the next nodes, if needed.
//

func (e *WorkflowNodeExecution) Pass(outputs map[string][]any) error {
	return e.PassInTransaction(database.Conn(), outputs)
}

func (e *WorkflowNodeExecution) PassInTransaction(tx *gorm.DB, outputs map[string][]any) error {
	now := time.Now()
	err := tx.Model(e).
		Updates(map[string]interface{}{
			"result":     WorkflowNodeExecutionResultPassed,
			"outputs":    datatypes.NewJSONType(outputs),
			"updated_at": &now,
		}).Error

	if err != nil {
		return err
	}

	return e.RouteInTransaction(tx)
}

func (e *WorkflowNodeExecution) Fail(reason, message string) error {
	return e.FailInTransaction(database.Conn(), reason, message)
}

func (e *WorkflowNodeExecution) FailInTransaction(tx *gorm.DB, reason, message string) error {
	now := time.Now()
	err := tx.Model(e).
		Updates(map[string]interface{}{
			"result":         WorkflowNodeExecutionResultFailed,
			"result_reason":  reason,
			"result_message": message,
			"updated_at":     &now,
		}).Error

	if err != nil {
		return err
	}

	return e.RouteInTransaction(tx)
}

func (e *WorkflowNodeExecution) Route() error {
	return e.RouteInTransaction(database.Conn())
}

func (e *WorkflowNodeExecution) RouteInTransaction(tx *gorm.DB) error {
	now := time.Now()
	e.State = WorkflowNodeExecutionStateRouting
	e.UpdatedAt = &now
	return tx.Save(e).Error
}

func (e *WorkflowNodeExecution) Finish() error {
	return e.FinishInTransaction(database.Conn())
}

func (e *WorkflowNodeExecution) FinishInTransaction(tx *gorm.DB) error {
	now := time.Now()
	return tx.Model(e).
		Updates(map[string]interface{}{
			"state":      WorkflowNodeExecutionStateFinished,
			"updated_at": &now,
		}).Error
}

func (e *WorkflowNodeExecution) GetInputs() (map[string]any, error) {
	//
	// First node in the flow - fetch from root event
	//
	if e.PreviousExecutionID == nil {
		initialEvent, err := FindWorkflowInitialEvent(e.RootEventID)
		if err != nil {
			return nil, fmt.Errorf("failed to find initial event %s: %w", e.RootEventID, err)
		}
		return initialEvent.Data.Data(), nil
	}

	previous, err := FindNodeExecution(*e.PreviousExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find previous execution %s: %w", *e.PreviousExecutionID, err)
	}

	//
	// Special case: Entering a blueprint
	// The previous execution is the top-level blueprint node execution,
	// so we need to get its inputs.
	//
	if e.ParentExecutionID != nil && previous.ParentExecutionID == nil {
		return previous.GetInputs()
	}

	if e.PreviousOutputBranch == nil || e.PreviousOutputIndex == nil {
		return nil, fmt.Errorf("execution %s has invalid previous reference", e.ID)
	}

	//
	// Normal case: read from previous execution's outputs,
	// using the output branch and index references.
	//
	previousOutputs := previous.Outputs.Data()
	branchData, exists := previousOutputs[*e.PreviousOutputBranch]
	if !exists {
		return nil, fmt.Errorf("previous execution %s has no output branch '%s'", previous.ID, *e.PreviousOutputBranch)
	}

	if *e.PreviousOutputIndex >= len(branchData) {
		return nil, fmt.Errorf("previous output index %d out of range for branch '%s'", *e.PreviousOutputIndex, *e.PreviousOutputBranch)
	}

	inputData, ok := branchData[*e.PreviousOutputIndex].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("previous output data is not a map")
	}

	return inputData, nil
}
