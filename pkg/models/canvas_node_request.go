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

type CanvasNodeRequest struct {
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

	now := time.Now()
	query := tx.
		Table("workflow_node_requests").
		Select("workflow_node_requests.*").
		Joins("JOIN workflow_nodes ON workflow_node_requests.workflow_id = workflow_nodes.workflow_id AND workflow_node_requests.node_id = workflow_nodes.node_id").
		Clauses(clause.Locking{
			Strength: "UPDATE",
			Table:    clause.Table{Name: "workflow_node_requests"},
			Options:  "SKIP LOCKED",
		}).
		Where("workflow_node_requests.id = ?", id).
		Where("workflow_node_requests.state = ?", NodeExecutionRequestStatePending).
		Where("workflow_node_requests.run_at <= ?", now).
		Where("workflow_nodes.deleted_at IS NULL")

	err := withActiveCanvas(query, "workflow_node_requests.workflow_id").
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
	query := database.Conn().
		Table("workflow_node_requests").
		Select("workflow_node_requests.*").
		Joins("JOIN workflow_nodes ON workflow_node_requests.workflow_id = workflow_nodes.workflow_id AND workflow_node_requests.node_id = workflow_nodes.node_id").
		Where("workflow_node_requests.state = ?", NodeExecutionRequestStatePending).
		Where("workflow_node_requests.run_at <= ?", now).
		Where("workflow_nodes.deleted_at IS NULL")

	err := withActiveCanvas(query, "workflow_node_requests.workflow_id").
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

func CountPendingRequestsForExecutionsInTransaction(tx *gorm.DB, executionIDs []uuid.UUID) (int64, error) {
	if len(executionIDs) == 0 {
		return 0, nil
	}

	var count int64
	err := tx.
		Model(&CanvasNodeRequest{}).
		Where("execution_id IN ?", executionIDs).
		Where("state = ?", NodeExecutionRequestStatePending).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}
