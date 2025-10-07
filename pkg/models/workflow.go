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
	WorkflowNodeExecutionStateWaiting  = "waiting"
	WorkflowNodeExecutionStateStarted  = "started"
	WorkflowNodeExecutionStateFinished = "finished"

	WorkflowNodeExecutionResultPassed = "passed"
	WorkflowNodeExecutionResultFailed = "failed"
)

type Workflow struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    string
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
	Nodes          datatypes.JSONSlice[Node]
	Edges          datatypes.JSONSlice[Edge]
}

func (w *Workflow) FindNode(id string) (*Node, error) {
	for _, node := range w.Nodes {
		if node.ID == id {
			return &node, nil
		}
	}

	return nil, fmt.Errorf("node %s not found", id)
}

type WorkflowQueueItem struct {
	WorkflowID uuid.UUID
	EventID    uuid.UUID
	NodeID     string
	CreatedAt  *time.Time
}

type WorkflowNodeExecution struct {
	ID            uuid.UUID
	WorkflowID    uuid.UUID
	NodeID        string
	EventID       uuid.UUID
	State         string
	Result        string
	ResultReason  string
	ResultMessage string
	Inputs        datatypes.JSONType[map[string]any]
	Outputs       datatypes.JSONType[map[string][]any]
	Metadata      datatypes.JSONType[map[string]any]
	CreatedAt     *time.Time
	UpdatedAt     *time.Time
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

func FindWorkflow(id uuid.UUID) (*Workflow, error) {
	var workflow Workflow
	err := database.Conn().
		Where("id = ?", id).
		First(&workflow).
		Error

	if err != nil {
		return nil, err
	}

	return &workflow, nil
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
