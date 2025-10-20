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
	Type          string
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

func (w *WorkflowNode) FirstQueueItem(tx *gorm.DB) (*WorkflowNodeQueueItem, error) {
	var queueItem WorkflowNodeQueueItem
	err := tx.
		Where("workflow_id = ?", w.WorkflowID).
		Where("node_id = ?", w.NodeID).
		Order("created_at ASC").
		First(&queueItem).
		Error

	if err != nil {
		return nil, err
	}

	return &queueItem, nil
}

type WorkflowNodeQueueItem struct {
	ID         uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	WorkflowID uuid.UUID
	NodeID     string
	CreatedAt  *time.Time

	//
	// Reference to the root WorkflowEvent record that started
	// this whole execution chain.
	//
	// This gives us an easy way to find all the queue items
	// for that event with a simple query.
	//
	RootEventID uuid.UUID

	//
	// The reference to a WorkflowEvent record,
	// which holds the input for this queue item.
	//
	EventID uuid.UUID
}

func (i *WorkflowNodeQueueItem) Delete(tx *gorm.DB) error {
	return tx.Delete(i).Error
}

func ListNodeQueueItems(workflowID uuid.UUID, nodeID string, limit int, beforeTime *time.Time) ([]WorkflowNodeQueueItem, error) {
	var queueItems []WorkflowNodeQueueItem
	query := database.Conn().
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Order("created_at DESC").
		Limit(int(limit))

	if beforeTime != nil {
		query = query.Where("created_at < ?", beforeTime)
	}

	err := query.Find(&queueItems).Error
	if err != nil {
		return nil, err
	}

	return queueItems, nil
}

func CountNodeQueueItems(workflowID uuid.UUID, nodeID string) (int64, error) {
	var totalCount int64
	countQuery := database.Conn().
		Model(&WorkflowNodeQueueItem{}).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID)

	if err := countQuery.Count(&totalCount).Error; err != nil {
		return 0, err
	}

	return totalCount, nil
}
