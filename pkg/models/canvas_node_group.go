package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NodeGroup is a drawn group of nodes on the canvas that acts as a queue.
// A run acquires a slot in the group when its first item dispatches into
// the group, and holds it while it has work inside.
type NodeGroup struct {
	ID    string   `json:"id"`
	Nodes []string `json:"nodes"`

	// Max is presence-aware: nil means the default (1).
	Max *int `json:"max,omitempty"`
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

// EffectiveMax returns the configured limit, defaulting to 1.
func (g *NodeGroup) EffectiveMax() int {
	if g == nil || g.Max == nil {
		return DefaultConcurrencyMax
	}
	return *g.Max
}

func (s *CanvasQueueSlot) TableName() string {
	return "workflow_queue_slots"
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
