package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	NodeRequestTypeInvokeAction = "invoke-action"

	NodeExecutionRequestStatePending   = "pending"
	NodeExecutionRequestStateCompleted = "completed"
)

type WorkflowNodeRequest struct {
	ID          uuid.UUID
	WorkflowID  uuid.UUID
	NodeID      string
	ExecutionID *uuid.UUID
	State       string
	Type        string
	Spec        datatypes.JSONType[NodeExecutionRequestSpec]
	RunAt       time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type NodeExecutionRequestSpec struct {
	InvokeAction *InvokeAction `json:"invoke_action,omitempty"`
}

type InvokeAction struct {
	ActionName string         `json:"action_name"`
	Parameters map[string]any `json:"parameters"`
}

func LockNodeRequest(tx *gorm.DB, id uuid.UUID) (*WorkflowNodeRequest, error) {
	var request WorkflowNodeRequest

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", id).
		First(&request).
		Error

	if err != nil {
		return nil, err
	}

	return &request, nil
}

func ListNodeRequests() ([]WorkflowNodeRequest, error) {
	var requests []WorkflowNodeRequest

	now := time.Now()
	err := database.Conn().
		Where("state = ?", NodeExecutionRequestStatePending).
		Where("run_at <= ?", now).
		Find(&requests).
		Error

	if err != nil {
		return nil, err
	}

	return requests, nil
}

func FindPendingRequestForNode(tx *gorm.DB, workflowID uuid.UUID, nodeID string) (*WorkflowNodeRequest, error) {
	var request WorkflowNodeRequest

	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Where("execution_id IS NULL").
		Where("state = ?", NodeExecutionRequestStatePending).
		First(&request).
		Error

	if err != nil {
		return nil, err
	}

	return &request, nil
}

func (r *WorkflowNodeRequest) Complete(tx *gorm.DB) error {
	return tx.Model(r).
		Update("state", NodeExecutionRequestStateCompleted).
		Update("updated_at", time.Now()).
		Error
}
