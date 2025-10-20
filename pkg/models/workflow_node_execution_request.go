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
	NodeExecutionRequestTypeInvokeAction = "invoke-action"

	NodeExecutionRequestStatePending   = "pending"
	NodeExecutionRequestStateCompleted = "completed"
)

type WorkflowNodeExecutionRequest struct {
	ID          uuid.UUID
	WorkflowID  uuid.UUID
	ExecutionID uuid.UUID
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

func LockNodeExecutionRequest(tx *gorm.DB, id uuid.UUID) (*WorkflowNodeExecutionRequest, error) {
	var request WorkflowNodeExecutionRequest

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

func ListNodeExecutionRequests() ([]WorkflowNodeExecutionRequest, error) {
	var requests []WorkflowNodeExecutionRequest

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

func (r *WorkflowNodeExecutionRequest) Complete(tx *gorm.DB) error {
	return tx.Model(r).
		Update("state", NodeExecutionRequestStateCompleted).
		Update("updated_at", time.Now()).
		Error
}
