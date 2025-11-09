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

	NodeTypeTrigger   = "trigger"
	NodeTypeComponent = "component"
	NodeTypeBlueprint = "blueprint"
)

type WorkflowNode struct {
	WorkflowID    uuid.UUID `gorm:"primaryKey"`
	NodeID        string    `gorm:"primaryKey"`
	ParentNodeID  *string
	Name          string
	State         string
	Type          string
	Position      datatypes.JSONType[Position]
	Ref           datatypes.JSONType[NodeRef]
	Configuration datatypes.JSONType[map[string]any]
	Metadata      datatypes.JSONType[map[string]any]
	IsCollapsed   bool
	WebhookID     *uuid.UUID
	CreatedAt     *time.Time
	UpdatedAt     *time.Time
}

func DeleteWorkflowNode(tx *gorm.DB, node WorkflowNode) error {
	err := tx.Delete(&node).Error
	if err != nil {
		return err
	}

	if node.WebhookID == nil {
		return nil
	}

	webhook, err := FindWebhookInTransaction(tx, *node.WebhookID)
	if err != nil {
		return err
	}

	return tx.Delete(&webhook).Error
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
		Where("type IN ?", []string{NodeTypeComponent, NodeTypeBlueprint}).
		Find(&nodes).
		Error

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func ListReadyTriggers() ([]WorkflowNode, error) {
	var nodes []WorkflowNode
	err := database.Conn().
		Where("state = ?", WorkflowNodeStateReady).
		Where("type = ?", NodeTypeTrigger).
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

func (w *WorkflowNode) CreateRequest(tx *gorm.DB, reqType string, spec NodeExecutionRequestSpec, runAt *time.Time) error {
	return tx.Create(&WorkflowNodeRequest{
		WorkflowID: w.WorkflowID,
		NodeID:     w.NodeID,
		ID:         uuid.New(),
		State:      NodeExecutionRequestStatePending,
		Type:       reqType,
		Spec:       datatypes.NewJSONType(spec),
		RunAt:      *runAt,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}).Error
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

// FindNextQueueItemPerNode finds the next (oldest) queue item for each node in a workflow
// using DISTINCT ON to get one queue item per node_id, ordered by created_at ASC
func FindNextQueueItemPerNode(workflowID uuid.UUID) ([]WorkflowNodeQueueItem, error) {
	var queueItems []WorkflowNodeQueueItem
	err := database.Conn().
		Raw(`
			SELECT DISTINCT ON (node_id) *
			FROM workflow_node_queue_items
			WHERE workflow_id = ?
			ORDER BY node_id, created_at ASC
		`, workflowID).
		Scan(&queueItems).
		Error

	if err != nil {
		return nil, err
	}

	return queueItems, nil
}
