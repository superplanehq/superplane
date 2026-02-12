package models

import (
	"fmt"
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

	RetryStrategyTypeConstant = "constant"
)

type CanvasNodeRequest struct {
	ID            uuid.UUID
	WorkflowID    uuid.UUID
	NodeID        string
	ExecutionID   *uuid.UUID
	State         string
	Type          string
	Spec          datatypes.JSONType[NodeExecutionRequestSpec]
	RetryStrategy datatypes.JSONType[RetryStrategy]
	Attempts      int
	RunAt         time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type RetryStrategy struct {
	Type     string
	Constant *ConstantRetryStrategy
}

func (r *RetryStrategy) NextRunAt() (*time.Time, error) {
	switch r.Type {
	case RetryStrategyTypeConstant:
		return r.Constant.NextRunAt()
	}

	return nil, fmt.Errorf("unknown retry type: %s", r.Type)
}

type ConstantRetryStrategy struct {
	MaxAttempts int
	Delay       time.Duration
}

func (r *ConstantRetryStrategy) NextRunAt() (*time.Time, error) {
	nextRunAt := time.Now().Add(r.Delay)
	return &nextRunAt, nil
}

func (r *CanvasNodeRequest) TableName() string {
	return "workflow_node_requests"
}

type NodeExecutionRequestSpec struct {
	InvokeAction *InvokeAction `json:"invoke_action,omitempty"`
}

type InvokeAction struct {
	ActionName string         `json:"action_name"`
	Parameters map[string]any `json:"parameters"`
}

func LockNodeRequest(tx *gorm.DB, id uuid.UUID) (*CanvasNodeRequest, error) {
	var request CanvasNodeRequest

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

func ListNodeRequests() ([]CanvasNodeRequest, error) {
	var requests []CanvasNodeRequest

	now := time.Now()
	err := database.Conn().
		Joins("JOIN workflow_nodes ON workflow_node_requests.workflow_id = workflow_nodes.workflow_id AND workflow_node_requests.node_id = workflow_nodes.node_id").
		Joins("JOIN workflows ON workflow_node_requests.workflow_id = workflows.id").
		Where("workflow_node_requests.state = ?", NodeExecutionRequestStatePending).
		Where("workflow_node_requests.run_at <= ?", now).
		Where("workflow_nodes.deleted_at IS NULL").
		Where("workflows.deleted_at IS NULL").
		Find(&requests).
		Error

	if err != nil {
		return nil, err
	}

	return requests, nil
}

func FindPendingRequestForNode(tx *gorm.DB, workflowID uuid.UUID, nodeID string) (*CanvasNodeRequest, error) {
	var request CanvasNodeRequest

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

func (r *CanvasNodeRequest) Complete(tx *gorm.DB) error {
	return tx.Model(r).
		Update("state", NodeExecutionRequestStateCompleted).
		Update("updated_at", time.Now()).
		Error
}
