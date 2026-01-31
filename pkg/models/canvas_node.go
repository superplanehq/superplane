package models

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CanvasNodeStateReady      = "ready"
	CanvasNodeStateProcessing = "processing"
	CanvasNodeStateError      = "error"
	CanvasNodeStatePaused     = "paused"

	NodeTypeTrigger   = "trigger"
	NodeTypeComponent = "component"
	NodeTypeBlueprint = "blueprint"
	NodeTypeWidget    = "widget"
)

type CanvasNode struct {
	WorkflowID        uuid.UUID `gorm:"primaryKey"`
	NodeID            string    `gorm:"primaryKey"`
	ParentNodeID      *string
	Name              string
	State             string
	StateReason       *string
	Type              string
	Position          datatypes.JSONType[Position]
	Ref               datatypes.JSONType[NodeRef]
	Configuration     datatypes.JSONType[map[string]any]
	Metadata          datatypes.JSONType[map[string]any]
	IsCollapsed       bool
	WebhookID         *uuid.UUID
	AppInstallationID *uuid.UUID
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

func (c *CanvasNode) TableName() string {
	return "workflow_nodes"
}

var nodeIDSanitizer = regexp.MustCompile(`[^a-z0-9]`)

func GenerateUniqueNodeID(node Node, reservedIDs map[string]bool) string {
	blockName := NodeTypeName(node)
	nodeName := node.Name
	if nodeName == "" {
		nodeName = "node"
	}

	for {
		candidate := GenerateNodeID(blockName, nodeName)
		if !reservedIDs[candidate] {
			return candidate
		}
	}
}

func GenerateNodeID(blockName string, nodeName string) string {
	randomChars := randomNodeSuffix(6)
	sanitizedBlock := sanitizeNodeIDSegment(blockName)
	sanitizedName := sanitizeNodeIDSegment(nodeName)
	return fmt.Sprintf("%s-%s-%s", sanitizedBlock, sanitizedName, randomChars)
}

func NodeTypeName(node Node) string {
	if node.Ref.Component != nil && node.Ref.Component.Name != "" {
		return node.Ref.Component.Name
	}
	if node.Ref.Trigger != nil && node.Ref.Trigger.Name != "" {
		return node.Ref.Trigger.Name
	}
	if node.Ref.Blueprint != nil && node.Ref.Blueprint.ID != "" {
		return node.Ref.Blueprint.ID
	}
	if node.Ref.Widget != nil && node.Ref.Widget.Name != "" {
		return node.Ref.Widget.Name
	}
	if node.Type != "" {
		return node.Type
	}
	return "node"
}

func sanitizeNodeIDSegment(value string) string {
	sanitized := nodeIDSanitizer.ReplaceAllString(strings.ToLower(value), "-")
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		return "node"
	}
	return sanitized
}

func randomNodeSuffix(length int) string {
	randomValue := strings.ReplaceAll(uuid.NewString(), "-", "")
	if length <= 0 {
		return randomValue
	}
	if length > len(randomValue) {
		return randomValue
	}
	return randomValue[:length]
}

func DeleteCanvasNode(tx *gorm.DB, node CanvasNode) error {
	err := tx.Delete(&node).Error
	if err != nil {
		return err
	}

	err = DeleteIntegrationSubscriptionsForNodeInTransaction(tx, node.WorkflowID, node.NodeID)
	if err != nil {
		return err
	}

	if node.WebhookID == nil {
		return nil
	}

	//
	// Delete the webhook associated with the node,
	// only if it does not have any other nodes associated with it.
	//
	webhook, err := FindWebhookInTransaction(tx, *node.WebhookID)
	if err != nil {
		return err
	}

	nodes, err := FindWebhookNodesInTransaction(tx, *node.WebhookID)
	if err != nil {
		return err
	}

	if len(nodes) > 0 {
		log.Printf("Webhook %s has %d other nodes associated with it", webhook.ID.String(), len(nodes))
		return nil
	}

	log.Printf("Deleting webhook %s", webhook.ID.String())
	return tx.Delete(&webhook).Error
}

func FindCanvasNode(tx *gorm.DB, canvasID uuid.UUID, nodeID string) (*CanvasNode, error) {
	var node CanvasNode
	err := tx.
		Where("workflow_id = ?", canvasID).
		Where("node_id = ?", nodeID).
		First(&node).
		Error

	if err != nil {
		return nil, err
	}

	return &node, nil
}

func FindCanvasNodesByIDs(tx *gorm.DB, canvasID uuid.UUID, nodeIDs []string) ([]CanvasNode, error) {
	var nodes []CanvasNode
	err := tx.
		Where("workflow_id = ? AND node_id IN ?", canvasID, nodeIDs).
		Find(&nodes).
		Error

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func ListCanvasNodesReady() ([]CanvasNode, error) {
	var nodes []CanvasNode
	err := database.Conn().
		Distinct().
		Joins("JOIN workflow_node_queue_items ON workflow_nodes.workflow_id = workflow_node_queue_items.workflow_id AND workflow_nodes.node_id = workflow_node_queue_items.node_id").
		Joins("JOIN workflows ON workflow_nodes.workflow_id = workflows.id").
		Where("workflow_nodes.state = ?", CanvasNodeStateReady).
		Where("workflow_nodes.type IN ?", []string{NodeTypeComponent, NodeTypeBlueprint}).
		Where("workflows.deleted_at IS NULL").
		Find(&nodes).
		Error

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func ListReadyTriggers() ([]CanvasNode, error) {
	var nodes []CanvasNode
	err := database.Conn().
		Where("state = ?", CanvasNodeStateReady).
		Where("type = ?", NodeTypeTrigger).
		Find(&nodes).
		Error

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func LockCanvasNode(tx *gorm.DB, workflowID uuid.UUID, nodeId string) (*CanvasNode, error) {
	var node CanvasNode

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeId).
		Where("state = ?", CanvasNodeStateReady).
		First(&node).
		Error

	if err != nil {
		return nil, err
	}

	return &node, nil
}

func LockCanvasNodeForUpdate(tx *gorm.DB, workflowID uuid.UUID, nodeId string) (*CanvasNode, error) {
	var node CanvasNode

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeId).
		First(&node).
		Error

	if err != nil {
		return nil, err
	}

	return &node, nil
}

func ResumeStateForNodeInTransaction(tx *gorm.DB, workflowID uuid.UUID, nodeID string) (string, error) {
	runningCount, err := CountRunningExecutionsForNodeInTransaction(tx, workflowID, nodeID)
	if err != nil {
		return "", err
	}

	if runningCount > 0 {
		return CanvasNodeStateProcessing, nil
	}

	return CanvasNodeStateReady, nil
}

func (w *CanvasNode) UpdateState(tx *gorm.DB, state string) error {
	return tx.Model(w).
		Update("state", state).
		Update("updated_at", time.Now()).
		Error
}

func (w *CanvasNode) FirstQueueItem(tx *gorm.DB) (*CanvasNodeQueueItem, error) {
	var queueItem CanvasNodeQueueItem
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

func (w *CanvasNode) CreateRequest(tx *gorm.DB, reqType string, spec NodeExecutionRequestSpec, runAt *time.Time) error {
	return tx.Create(&CanvasNodeRequest{
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

type CanvasNodeQueueItem struct {
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
	RootEvent   *CanvasEvent `gorm:"foreignKey:RootEventID"`

	//
	// The reference to a CanvasEvent record,
	// which holds the input for this queue item.
	//
	EventID uuid.UUID
}

func (i *CanvasNodeQueueItem) TableName() string {
	return "workflow_node_queue_items"
}

func (i *CanvasNodeQueueItem) Delete(tx *gorm.DB) error {
	return tx.Delete(i).Error
}

func ListNodeQueueItems(workflowID uuid.UUID, nodeID string, limit int, beforeTime *time.Time) ([]CanvasNodeQueueItem, error) {
	var queueItems []CanvasNodeQueueItem
	query := database.Conn().
		Preload("RootEvent").
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
		Model(&CanvasNodeQueueItem{}).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID)

	if err := countQuery.Count(&totalCount).Error; err != nil {
		return 0, err
	}

	return totalCount, nil
}

// FindNextQueueItemPerNode finds the next (oldest) queue item for each node in a workflow
// using DISTINCT ON to get one queue item per node_id, ordered by created_at ASC
// Only returns queue items for nodes that have not been deleted
func FindNextQueueItemPerNode(workflowID uuid.UUID) ([]CanvasNodeQueueItem, error) {
	var queueItems []CanvasNodeQueueItem
	err := database.Conn().
		Raw(`
			SELECT DISTINCT ON (qi.node_id) qi.*
			FROM workflow_node_queue_items qi
			INNER JOIN workflow_nodes wn
				ON qi.workflow_id = wn.workflow_id
				AND qi.node_id = wn.node_id
			WHERE qi.workflow_id = ?
			AND wn.deleted_at IS NULL
			ORDER BY qi.node_id, qi.created_at ASC
		`, workflowID).
		Scan(&queueItems).
		Error

	if err != nil {
		return nil, err
	}

	return queueItems, nil
}

func FindNodeQueueItem(workflowID uuid.UUID, queueItemID uuid.UUID) (*CanvasNodeQueueItem, error) {
	var queueItem CanvasNodeQueueItem
	err := database.Conn().
		Preload("RootEvent").
		Where("workflow_id = ? AND id = ?", workflowID, queueItemID).
		First(&queueItem).
		Error

	if err != nil {
		return nil, err
	}

	return &queueItem, nil
}
