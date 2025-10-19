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
	NodeExecutionRequestTypeQueueCheck   = "queue-check"

	NodeExecutionRequestStatePending   = "pending"
	NodeExecutionRequestStateCompleted = "completed"
)

type NodeExecutionRequest struct {
	ID          uuid.UUID
	WorkflowID  uuid.UUID
	ExecutionID uuid.UUID
	State       string
	Type        string
	Spec        datatypes.JSONType[NodeExecutionRequestSpec]
	RunAt       *time.Time
	CreatedAt   time.Time
}

type NodeExecutionRequestSpec struct {
	InvokeAction *InvokeAction `json:"invoke_action,omitempty"`
	QueueCheck   *QueueCheck   `json:"queue_check,omitempty"`
}

type InvokeAction struct {
	ActionName string         `json:"action_name"`
	Parameters map[string]any `json:"parameters"`
}

type QueueCheck struct {
	ActionName string `json:"action_name"`
}

func LockNodeExecutionRequest(tx *gorm.DB, id uuid.UUID) (*NodeExecutionRequest, error) {
	var request NodeExecutionRequest

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

func ListNodeExecutionRequests() ([]NodeExecutionRequest, error) {
	var requests []NodeExecutionRequest

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

func (r *NodeExecutionRequest) Complete(tx *gorm.DB) error {
	return tx.Model(r).
		Update("state", NodeExecutionRequestStateCompleted).
		Error
}
