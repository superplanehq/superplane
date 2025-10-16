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
	WorkflowNodeStateReady      = "ready"
	WorkflowNodeStateProcessing = "processing"
)

type WorkflowNode struct {
	WorkflowID    uuid.UUID `gorm:"primaryKey"`
	NodeID        string    `gorm:"primaryKey"`
	Name          string
	State         string
	RefType       string
	Ref           datatypes.JSONType[NodeRef]
	Configuration datatypes.JSONType[map[string]any]
	CreatedAt     *time.Time
	UpdatedAt     *time.Time
}

func DeleteWorkflowNode(tx *gorm.DB, workflowID uuid.UUID, nodeID string) error {
	return tx.
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Delete(&WorkflowNode{}).
		Error
}

func FindWorkflowNode(tx *gorm.DB, workflowID uuid.UUID, nodeID string) (*WorkflowNode, error) {
	var node WorkflowNode
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		First(&node).
		Error

	if err != nil {
		return nil, err
	}

	return &node, nil
}

func ListWorkflowNodesReady() ([]WorkflowNode, error) {
	var nodes []WorkflowNode
	err := database.Conn().
		Where("state = ?", WorkflowNodeStateReady).
		Find(&nodes).
		Error

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func LockWorkflowNode(tx *gorm.DB, workflowID uuid.UUID, nodeId string) (*WorkflowNode, error) {
	var node WorkflowNode

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeId).
		Where("state = ?", WorkflowNodeStateReady).
		First(&node).
		Error

	if err != nil {
		return nil, err
	}

	return &node, nil
}

func (w *WorkflowNode) UpdateState(tx *gorm.DB, state string) error {
	return tx.Model(w).
		Update("state", state).
		Update("updated_at", time.Now()).
		Error
}

func (w *WorkflowNode) FirstPendingExecution(tx *gorm.DB) (*WorkflowNodeExecution, error) {
	var execution WorkflowNodeExecution
	err := tx.
		Where("workflow_id = ?", w.WorkflowID).
		Where("node_id = ?", w.NodeID).
		Where("state = ?", WorkflowNodeExecutionStatePending).
		Order("created_at ASC").
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}
