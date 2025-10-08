package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type WorkflowNodeExecution struct {
	ID            uuid.UUID
	WorkflowID    uuid.UUID
	NodeID        string
	EventID       uuid.UUID
	State         string
	Result        string
	ResultReason  string
	ResultMessage string

	//
	// The event data (if first node in the flow),
	// or the output of the previous execution.
	//
	Inputs datatypes.JSONType[map[string]any]

	//
	// The outputs of the node execution.
	// Note that this is a map[string][]any type.
	// The key in the map is the output branch name.
	// A node can emit multiple events as part of the same output.
	// The subsequent node in the flow will unpack that and create
	// multiple queue items - and subsequent node executions.
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
	err := database.Conn().
		Where("state = ?", WorkflowNodeExecutionStatePending).
		Find(&executions).
		Error

	if err != nil {
		return nil, err
	}

	return executions, nil
}

func FindLastNodeExecution(workflowID uuid.UUID, states []string) (*WorkflowNodeExecution, error) {
	var execution WorkflowNodeExecution
	err := database.Conn().
		Where("workflow_id = ?", workflowID).
		Where("state IN ?", states).
		Order("updated_at DESC").
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func FindLastNodeExecutionForEvent(eventID uuid.UUID, states []string) (*WorkflowNodeExecution, error) {
	var execution WorkflowNodeExecution
	err := database.Conn().
		Where("event_id = ?", eventID).
		Where("state IN ?", states).
		Order("updated_at DESC").
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

func (e *WorkflowNodeExecution) Pass(outputs map[string][]any) error {
	return e.PassInTransaction(database.Conn(), outputs)
}

func (e *WorkflowNodeExecution) Wait() error {
	return e.WaitInTransaction(database.Conn())
}

func (e *WorkflowNodeExecution) WaitInTransaction(tx *gorm.DB) error {
	now := time.Now()
	e.State = WorkflowNodeExecutionStateWaiting
	e.UpdatedAt = &now
	return tx.Save(e).Error
}

func (e *WorkflowNodeExecution) PassInTransaction(tx *gorm.DB, outputs map[string][]any) error {
	now := time.Now()
	return tx.Model(e).
		Updates(map[string]interface{}{
			"state":      WorkflowNodeExecutionStateFinished,
			"result":     WorkflowNodeExecutionResultPassed,
			"outputs":    datatypes.NewJSONType(outputs),
			"updated_at": &now,
		}).Error
}

func (e *WorkflowNodeExecution) Fail(reason string) error {
	return e.FailInTransaction(database.Conn(), reason)
}

func (e *WorkflowNodeExecution) FailInTransaction(tx *gorm.DB, reason string) error {
	now := time.Now()
	return tx.Model(e).
		Updates(map[string]interface{}{
			"state":         WorkflowNodeExecutionStateFinished,
			"result":        WorkflowNodeExecutionResultFailed,
			"result_reason": reason,
			"updated_at":    &now,
		}).Error
}
