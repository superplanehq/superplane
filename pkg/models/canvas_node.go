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

	NodeTypeTrigger   = "trigger"
	NodeTypeComponent = "component"
	NodeTypeWidget    = "widget"
)

type Node struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	Ref            NodeRef        `json:"ref"`
	Configuration  map[string]any `json:"configuration"`
	Metadata       map[string]any `json:"metadata"`
	Position       Position       `json:"position"`
	IsCollapsed    bool           `json:"isCollapsed"`
	IntegrationID  *string        `json:"integrationId,omitempty"`
	ErrorMessage   *string        `json:"errorMessage,omitempty"`
	WarningMessage *string        `json:"warningMessage,omitempty"`
}

func (c *Node) ComponentName() string {
	if c.Ref.Component != nil && c.Ref.Component.Name != "" {
		return c.Ref.Component.Name
	}

	if c.Ref.Trigger != nil && c.Ref.Trigger.Name != "" {
		return c.Ref.Trigger.Name
	}

	if c.Ref.Widget != nil && c.Ref.Widget.Name != "" {
		return c.Ref.Widget.Name
	}

	return ""
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type NodeRef struct {
	Component *ComponentRef `json:"component,omitempty"`
	Trigger   *TriggerRef   `json:"trigger,omitempty"`
	Widget    *WidgetRef    `json:"widget,omitempty"`
}

type ComponentRef struct {
	Name string `json:"name"`
}

type TriggerRef struct {
	Name string `json:"name"`
}

type WidgetRef struct {
	Name string `json:"name"`
}

type Edge struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Channel  string `json:"channel"`
}

type CanvasNode struct {
	WorkflowID        uuid.UUID `gorm:"primaryKey"`
	NodeID            string    `gorm:"primaryKey"`
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

type DeleteCanvasNodeResult struct {
	CancelledExecutionIDs []uuid.UUID
	DeletedQueueItems     []CanvasNodeQueueItem
}

func (c *CanvasNode) TableName() string {
	return "workflow_nodes"
}

func (c *CanvasNode) ComponentName() string {
	ref := c.Ref.Data()
	if ref.Component != nil && ref.Component.Name != "" {
		return ref.Component.Name
	}

	if ref.Trigger != nil && ref.Trigger.Name != "" {
		return ref.Trigger.Name
	}

	return "unknown"
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
	_, err := DeleteCanvasNodeWithResult(tx, node)
	return err
}

func DeleteCanvasNodeWithResult(tx *gorm.DB, node CanvasNode) (DeleteCanvasNodeResult, error) {
	result, err := cancelActiveExecutionsForDeletedNode(tx, node.WorkflowID, node.NodeID)
	if err != nil {
		return DeleteCanvasNodeResult{}, err
	}

	err = tx.Delete(&node).Error
	if err != nil {
		return DeleteCanvasNodeResult{}, err
	}

	err = DeleteIntegrationSubscriptionsForNodeInTransaction(tx, node.WorkflowID, node.NodeID)
	if err != nil {
		return DeleteCanvasNodeResult{}, err
	}

	err = DeleteCanvasSubscriptionsForNode(tx, node.WorkflowID, node.NodeID)
	if err != nil {
		return DeleteCanvasNodeResult{}, err
	}

	if node.WebhookID == nil {
		return result, nil
	}

	//
	// Delete the webhook associated with the node,
	// only if it does not have any other nodes associated with it.
	//
	webhook, err := FindWebhookInTransaction(tx, *node.WebhookID)
	if err != nil {
		return DeleteCanvasNodeResult{}, err
	}

	nodes, err := FindWebhookNodesInTransaction(tx, *node.WebhookID)
	if err != nil {
		return DeleteCanvasNodeResult{}, err
	}

	if len(nodes) > 0 {
		log.Printf("Webhook %s has %d other nodes associated with it", webhook.ID.String(), len(nodes))
		return result, nil
	}

	log.Printf("Deleting webhook %s", webhook.ID.String())
	return result, tx.Delete(&webhook).Error
}

func (c *CanvasNode) HardDelete(tx *gorm.DB) error {
	return tx.Unscoped().
		Where("workflow_id = ? AND node_id = ?", c.WorkflowID, c.NodeID).
		Delete(c).
		Error
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

func FindUnscopedCanvasNode(tx *gorm.DB, canvasID uuid.UUID, nodeID string) (*CanvasNode, error) {
	var node CanvasNode
	err := tx.Unscoped().
		Where("workflow_id = ?", canvasID).
		Where("node_id = ?", nodeID).
		First(&node).
		Error

	if err != nil {
		return nil, err
	}

	return &node, nil
}

// ListDeletedCanvasNodes returns soft-deleted nodes whose parent canvas and
// organization are still active, and whose deleted_at is on or before before
// (typically now minus the cleanup grace period). Nodes on soft-deleted canvases
// or organizations are owned by CanvasCleanupWorker / OrganizationCleanupWorker.
// Results are capped by limit. Ordering prefers nodes that have waited longest
// for a cleanup pass (updated_at), then oldest soft-delete time, so nodes that
// cannot make progress can be rotated to the back of the queue.
func ListDeletedCanvasNodes(tx *gorm.DB, before time.Time, limit int) ([]CanvasNode, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive")
	}

	var nodes []CanvasNode
	query := tx.Unscoped().
		Model(&CanvasNode{}).
		Where("workflow_nodes.deleted_at IS NOT NULL").
		Where("workflow_nodes.deleted_at <= ?", before.UTC()).
		Order("workflow_nodes.updated_at ASC NULLS FIRST, workflow_nodes.deleted_at ASC").
		Limit(limit)

	err := withActiveCanvas(query, "workflow_nodes.workflow_id").
		Find(&nodes).
		Error
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// RotateCleanupQueue bumps updated_at so a blocked soft-deleted node yields the
// front of ListDeletedCanvasNodes to other candidates.
func (c *CanvasNode) RotateCleanupQueue(tx *gorm.DB) error {
	now := time.Now().UTC()
	return tx.Unscoped().Model(c).
		Where("workflow_id = ? AND node_id = ?", c.WorkflowID, c.NodeID).
		Update("updated_at", now).Error
}

// LockDeletedCanvasNode acquires a row-level lock on a soft-deleted canvas node
// whose parent canvas and organization are still active.
func LockDeletedCanvasNode(tx *gorm.DB, workflowID uuid.UUID, nodeID string) (*CanvasNode, error) {
	var node CanvasNode
	query := tx.Unscoped().
		Table("workflow_nodes").
		Select("workflow_nodes.*").
		Clauses(clause.Locking{
			Strength: "UPDATE",
			Table:    clause.Table{Name: "workflow_nodes"},
			Options:  "SKIP LOCKED",
		}).
		Where("workflow_nodes.workflow_id = ?", workflowID).
		Where("workflow_nodes.node_id = ?", nodeID).
		Where("workflow_nodes.deleted_at IS NOT NULL")

	err := withActiveCanvas(query, "workflow_nodes.workflow_id").
		First(&node).
		Error
	if err != nil {
		return nil, err
	}

	return &node, nil
}

func FindCanvasNodesByIDs(tx *gorm.DB, canvasID uuid.UUID, nodeIDs []string) ([]CanvasNode, error) {
	if len(nodeIDs) == 0 {
		return nil, nil
	}

	var nodes []CanvasNode
	err := tx.
		Joins("JOIN workflows ON workflows.id = workflow_nodes.workflow_id AND workflows.deleted_at IS NULL").
		Where("workflow_nodes.workflow_id = ? AND workflow_nodes.node_id IN ?", canvasID, nodeIDs).
		Find(&nodes).
		Error

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func ListCanvasNodesReady() ([]CanvasNode, error) {
	var nodes []CanvasNode
	query := database.Conn().
		Distinct().
		Joins("JOIN workflow_node_queue_items ON workflow_nodes.workflow_id = workflow_node_queue_items.workflow_id AND workflow_nodes.node_id = workflow_node_queue_items.node_id").
		Where("workflow_nodes.state = ?", CanvasNodeStateReady).
		Where("workflow_nodes.type = ?", NodeTypeComponent).
		Where("workflow_nodes.deleted_at IS NULL")

	err := withActiveCanvas(query, "workflow_nodes.workflow_id").
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

	query := tx.
		Table("workflow_nodes").
		Select("workflow_nodes.*").
		Clauses(clause.Locking{
			Strength: lockingForUpdateNoKey,
			Table:    clause.Table{Name: "workflow_nodes"},
			Options:  "SKIP LOCKED",
		}).
		Where("workflow_nodes.workflow_id = ?", workflowID).
		Where("workflow_nodes.node_id = ?", nodeId).
		Where("workflow_nodes.state = ?", CanvasNodeStateReady).
		Where("workflow_nodes.deleted_at IS NULL")

	err := withActiveCanvas(query, "workflow_nodes.workflow_id").
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
		Clauses(clause.Locking{Strength: lockingForUpdateNoKey, Options: "SKIP LOCKED"}).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeId).
		First(&node).
		Error

	if err != nil {
		return nil, err
	}

	return &node, nil
}

func (c *CanvasNode) UpdateState(tx *gorm.DB, state string) error {
	return tx.Model(c).
		Update("state", state).
		Update("updated_at", time.Now()).
		Error
}

func (c *CanvasNode) FirstQueueItem(tx *gorm.DB) (*CanvasNodeQueueItem, error) {
	var queueItem CanvasNodeQueueItem
	err := tx.
		Where("workflow_id = ?", c.WorkflowID).
		Where("node_id = ?", c.NodeID).
		Order("created_at ASC").
		First(&queueItem).
		Error

	if err != nil {
		return nil, err
	}

	return &queueItem, nil
}

func (c *CanvasNode) CreateRequest(tx *gorm.DB, reqType string, spec NodeExecutionRequestSpec, runAt *time.Time) error {
	return tx.Create(&CanvasNodeRequest{
		WorkflowID: c.WorkflowID,
		NodeID:     c.NodeID,
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
	RunID       uuid.UUID

	//
	// The reference to a CanvasEvent record,
	// which holds the input for this queue item.
	//
	EventID uuid.UUID
}

func (i *CanvasNodeQueueItem) TableName() string {
	return "workflow_node_queue_items"
}

func (i *CanvasNodeQueueItem) BeforeCreate(tx *gorm.DB) error {
	if i.RunID != uuid.Nil {
		return nil
	}

	run, err := FindCanvasRunByRootEventInTransaction(tx, i.RootEventID)
	if err != nil {
		return err
	}

	i.RunID = run.ID
	return nil
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

func CountNodeQueueItemsForRootEventInTransaction(tx *gorm.DB, rootEventID uuid.UUID) (int64, error) {
	var count int64

	err := tx.
		Model(&CanvasNodeQueueItem{}).
		Where("root_event_id = ?", rootEventID).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func ListNodeQueueItemsForRuns(tx *gorm.DB, workflowID uuid.UUID, runIDs []uuid.UUID) ([]CanvasNodeQueueItem, error) {
	if len(runIDs) == 0 {
		return []CanvasNodeQueueItem{}, nil
	}

	var queueItems []CanvasNodeQueueItem
	err := tx.
		Preload("RootEvent").
		Where("workflow_id = ?", workflowID).
		Where("run_id IN ?", runIDs).
		Order("created_at ASC").
		Find(&queueItems).
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

func cancelActiveExecutionsForDeletedNode(tx *gorm.DB, workflowID uuid.UUID, nodeID string) (DeleteCanvasNodeResult, error) {
	executions, err := ListActiveNodeExecutions(tx, workflowID, nodeID)
	if err != nil {
		return DeleteCanvasNodeResult{}, err
	}

	cancelledExecutionIDs, err := cancelNodeExecutions(tx, executions, nil)
	if err != nil {
		return DeleteCanvasNodeResult{}, err
	}

	deletedQueueItems, err := deleteQueueItemsForNode(tx, workflowID, nodeID)
	if err != nil {
		return DeleteCanvasNodeResult{}, err
	}

	if err := completePendingRequestsForNodeExecutions(tx, workflowID, nodeID); err != nil {
		return DeleteCanvasNodeResult{}, err
	}

	return DeleteCanvasNodeResult{
		CancelledExecutionIDs: cancelledExecutionIDs,
		DeletedQueueItems:     deletedQueueItems,
	}, nil
}

func deleteQueueItemsForNode(tx *gorm.DB, workflowID uuid.UUID, nodeID string) ([]CanvasNodeQueueItem, error) {
	var deletedQueueItems []CanvasNodeQueueItem
	err := tx.
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}, {Name: "node_id"}, {Name: "run_id"}, {Name: "workflow_id"}}}).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Delete(&deletedQueueItems).
		Error
	if err != nil {
		return nil, err
	}

	return deletedQueueItems, nil
}

func cancelNodeExecutions(tx *gorm.DB, executions []CanvasNodeExecution, cancelledBy *uuid.UUID) ([]uuid.UUID, error) {
	requestedCancellationIDs := make([]uuid.UUID, 0, len(executions))

	for i := range executions {
		execution := executions[i]
		if execution.State == CanvasNodeExecutionStateFinished || execution.State == CanvasNodeExecutionStateCancelling {
			continue
		}

		if err := execution.RequestCancellation(tx, cancelledBy); err != nil {
			return nil, err
		}

		requestedCancellationIDs = append(requestedCancellationIDs, execution.ID)
	}

	return requestedCancellationIDs, nil
}

func completePendingRequestsForNodeExecutions(tx *gorm.DB, workflowID uuid.UUID, nodeID string) error {
	now := time.Now()
	return tx.
		Model(&CanvasNodeRequest{}).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Where("execution_id IS NOT NULL").
		Where("state = ?", NodeExecutionRequestStatePending).
		Updates(map[string]any{
			"state":      NodeExecutionRequestStateCompleted,
			"updated_at": now,
		}).
		Error
}
