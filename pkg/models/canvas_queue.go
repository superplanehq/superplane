package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	QueueAutoCancelQueued  = "queued"
	QueueAutoCancelRunning = "running"

	DefaultQueueMaxParallelism = 1
)

// QueueSpec is a node's inline queue configuration. A node without a
// spec uses its implicit queue: named after the node ID, maxParallelism 1.
type QueueSpec struct {
	// Key is an optional template expression that partitions the node's
	// backlog: each resolved value is an independent queue.
	Key string `json:"key,omitempty"`

	// MaxParallelism is presence-aware: nil means the default (1), and 0
	// means unlimited, which disables queueing for the node entirely.
	MaxParallelism *int `json:"maxParallelism,omitempty"`

	AutoCancel string `json:"autoCancel,omitempty"`
}

// EffectiveMaxParallelism returns the configured limit, defaulting to 1
// when the spec or the field is absent. 0 means unlimited.
func (s *QueueSpec) EffectiveMaxParallelism() int {
	if s == nil || s.MaxParallelism == nil {
		return DefaultQueueMaxParallelism
	}
	return *s.MaxParallelism
}

// Unlimited reports whether the spec disables queueing (maxParallelism 0).
func (s *QueueSpec) Unlimited() bool {
	return s.EffectiveMaxParallelism() == 0
}

func (s *QueueSpec) AutoCancelPolicy() string {
	if s == nil {
		return ""
	}
	return s.AutoCancel
}

// NodeGroup is a drawn group of nodes on the canvas that acts as a queue.
// A run acquires a slot in the group when its first item dispatches into
// the group, and holds it while it has work inside.
type NodeGroup struct {
	ID    string   `json:"id"`
	Nodes []string `json:"nodes"`

	// MaxParallelism is presence-aware: nil means the default (1).
	MaxParallelism *int `json:"maxParallelism,omitempty"`
}

// EffectiveMaxParallelism returns the configured limit, defaulting to 1.
func (g *NodeGroup) EffectiveMaxParallelism() int {
	if g == nil || g.MaxParallelism == nil {
		return DefaultQueueMaxParallelism
	}
	return *g.MaxParallelism
}

// CanvasQueueSlot records a run holding a slot in a group.
// Node-queue capacity is derived from execution counts and never
// creates rows here.
type CanvasQueueSlot struct {
	WorkflowID uuid.UUID `gorm:"primaryKey"`
	GroupID    string    `gorm:"primaryKey"`
	RunID      uuid.UUID `gorm:"primaryKey"`
	AcquiredAt *time.Time
}

func (s *CanvasQueueSlot) TableName() string {
	return "workflow_queue_slots"
}

// SyncCanvasNodeGroupIDs materializes node groups onto
// workflow_nodes.group_id, so the queue worker can gate dispatch without
// loading the canvas version.
func SyncCanvasNodeGroupIDs(tx *gorm.DB, workflowID uuid.UUID, groups []NodeGroup) error {
	groupedNodeIDs := make([]string, 0)
	for _, group := range groups {
		err := tx.Model(&CanvasNode{}).
			Where("workflow_id = ?", workflowID).
			Where("node_id IN ?", group.Nodes).
			Update("group_id", group.ID).
			Error
		if err != nil {
			return err
		}

		groupedNodeIDs = append(groupedNodeIDs, group.Nodes...)
	}

	query := tx.Model(&CanvasNode{}).
		Where("workflow_id = ?", workflowID).
		Where("group_id IS NOT NULL")
	if len(groupedNodeIDs) > 0 {
		query = query.Where("node_id NOT IN ?", groupedNodeIDs)
	}

	return query.Update("group_id", nil).Error
}

// CountActiveExecutionsInQueue counts the executions occupying slots in a
// node's queue. Queues are private to a node, so capacity is scoped by
// (workflow, node, resolved queue name).
func CountActiveExecutionsInQueue(tx *gorm.DB, workflowID uuid.UUID, nodeID, queueName string) (int64, error) {
	var count int64
	err := tx.
		Model(&CanvasNodeExecution{}).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Where("queue_name = ?", queueName).
		Where("state IN ?", CanvasNodeExecutionActiveStates).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func ListActiveExecutionsInQueue(tx *gorm.DB, workflowID uuid.UUID, nodeID, queueName string) ([]CanvasNodeExecution, error) {
	var executions []CanvasNodeExecution
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Where("queue_name = ?", queueName).
		Where("state IN ?", CanvasNodeExecutionActiveStates).
		Order("created_at ASC").
		Find(&executions).
		Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

// FindQueueSlotForRun returns the group-queue slot held by a run, or nil
// when the run holds none. A run holds at most one slot at a time.
func FindQueueSlotForRun(tx *gorm.DB, workflowID, runID uuid.UUID) (*CanvasQueueSlot, error) {
	var slot CanvasQueueSlot
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("run_id = ?", runID).
		First(&slot).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &slot, nil
}

func CountQueueSlots(tx *gorm.DB, workflowID uuid.UUID, groupID string) (int64, error) {
	var count int64
	err := tx.
		Model(&CanvasQueueSlot{}).
		Where("workflow_id = ?", workflowID).
		Where("group_id = ?", groupID).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func AcquireQueueSlot(tx *gorm.DB, workflowID uuid.UUID, groupID string, runID uuid.UUID) error {
	now := time.Now()
	return tx.Create(&CanvasQueueSlot{
		WorkflowID: workflowID,
		GroupID:    groupID,
		RunID:      runID,
		AcquiredAt: &now,
	}).Error
}

func DeleteQueueSlotsForRun(tx *gorm.DB, workflowID, runID uuid.UUID) error {
	return tx.
		Where("workflow_id = ?", workflowID).
		Where("run_id = ?", runID).
		Delete(&CanvasQueueSlot{}).
		Error
}

// ReleaseQueueSlotIfGroupIdle frees the group-queue slot held by a run
// when the run has no work left inside the group: no pending queue items
// and no active executions on the group's nodes. It returns the released
// slot so callers can wake up waiters, or nil when nothing was released.
func ReleaseQueueSlotIfGroupIdle(tx *gorm.DB, workflowID uuid.UUID, groupID string, runID uuid.UUID) (*CanvasQueueSlot, error) {
	var slot CanvasQueueSlot
	err := tx.
		Where("workflow_id = ?", workflowID).
		Where("group_id = ?", groupID).
		Where("run_id = ?", runID).
		First(&slot).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	groupNodeIDs, err := listGroupNodeIDs(tx, workflowID, groupID)
	if err != nil {
		return nil, err
	}

	var pendingItems int64
	err = tx.
		Model(&CanvasNodeQueueItem{}).
		Where("workflow_id = ?", workflowID).
		Where("run_id = ?", runID).
		Where("node_id IN ?", groupNodeIDs).
		Count(&pendingItems).
		Error
	if err != nil {
		return nil, err
	}

	if pendingItems > 0 {
		return nil, nil
	}

	var activeExecutions int64
	err = tx.
		Model(&CanvasNodeExecution{}).
		Where("workflow_id = ?", workflowID).
		Where("run_id = ?", runID).
		Where("node_id IN ?", groupNodeIDs).
		Where("state IN ?", CanvasNodeExecutionActiveStates).
		Count(&activeExecutions).
		Error
	if err != nil {
		return nil, err
	}

	if activeExecutions > 0 {
		return nil, nil
	}

	//
	// Pending (unrouted) events emitted by group nodes may still route to
	// other nodes inside the group, so they count as work-in-group. The
	// event router re-runs this check after routing them.
	//
	var pendingEvents int64
	err = tx.
		Model(&CanvasEvent{}).
		Where("workflow_id = ?", workflowID).
		Where("run_id = ?", runID).
		Where("node_id IN ?", groupNodeIDs).
		Where("state = ?", CanvasEventStatePending).
		Count(&pendingEvents).
		Error
	if err != nil {
		return nil, err
	}

	if pendingEvents > 0 {
		return nil, nil
	}

	err = tx.
		Where("workflow_id = ?", slot.WorkflowID).
		Where("group_id = ?", slot.GroupID).
		Where("run_id = ?", slot.RunID).
		Delete(&CanvasQueueSlot{}).
		Error
	if err != nil {
		return nil, err
	}

	return &slot, nil
}

// FindLiveCanvasNodeGroups loads the node groups from the live canvas
// version. A canvas without a live version has no groups.
func FindLiveCanvasNodeGroups(tx *gorm.DB, workflowID uuid.UUID) ([]NodeGroup, error) {
	version, err := FindLiveCanvasVersionInTransaction(tx, workflowID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return append([]NodeGroup(nil), version.NodeGroups...), nil
}

// SupersedeQueueItem removes a queue item that was replaced by a newer
// item in a queue with autoCancel: queued. When the item was the run's
// only remaining work, the run finishes with the superseded result.
func SupersedeQueueItem(tx *gorm.DB, item *CanvasNodeQueueItem) error {
	if err := tx.Delete(item).Error; err != nil {
		return err
	}

	run, err := FindCanvasRunInTransaction(tx, item.WorkflowID, item.RunID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	if run.State == CanvasRunStateFinished {
		return nil
	}

	openWork, err := run.FindOpenWork(tx)
	if err != nil {
		return err
	}

	if openWork.HasActiveExecutions || openWork.HasQueueItems || openWork.HasPendingEvents {
		return nil
	}

	now := time.Now()
	err = tx.Model(run).
		Updates(map[string]any{
			"state":       CanvasRunStateFinished,
			"result":      CanvasRunResultSuperseded,
			"updated_at":  &now,
			"finished_at": &now,
		}).
		Error
	if err != nil {
		return err
	}

	return DeleteQueueSlotsForRun(tx, run.WorkflowID, run.ID)
}

// ListWaitingQueueItemsForGroup returns the oldest waiting queue item per
// node of a group, across all runs. Used to wake up waiters after a
// group-queue slot is released.
func ListWaitingQueueItemsForGroup(tx *gorm.DB, workflowID uuid.UUID, groupID string) ([]CanvasNodeQueueItem, error) {
	groupNodeIDs, err := listGroupNodeIDs(tx, workflowID, groupID)
	if err != nil {
		return nil, err
	}

	if len(groupNodeIDs) == 0 {
		return nil, nil
	}

	var items []CanvasNodeQueueItem
	err = tx.
		Raw(`
			SELECT DISTINCT ON (node_id) *
			FROM workflow_node_queue_items
			WHERE workflow_id = ? AND node_id IN ?
			ORDER BY node_id, created_at ASC
		`, workflowID, groupNodeIDs).
		Scan(&items).
		Error
	if err != nil {
		return nil, err
	}

	return items, nil
}

func listGroupNodeIDs(tx *gorm.DB, workflowID uuid.UUID, groupID string) ([]string, error) {
	var nodeIDs []string
	err := tx.
		Model(&CanvasNode{}).
		Where("workflow_id = ?", workflowID).
		Where("group_id = ?", groupID).
		Where("deleted_at IS NULL").
		Pluck("node_id", &nodeIDs).
		Error
	if err != nil {
		return nil, err
	}

	return nodeIDs, nil
}
