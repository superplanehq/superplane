package models

import (
	"fmt"

	"gorm.io/gorm"
)

type nodeResourceEventMode int

const (
	nodeResourceEventModeUnset nodeResourceEventMode = iota
	nodeResourceEventModeAll
	nodeResourceEventModeUnreferenced
)

type NodeResourceCleaner struct {
	tx        *gorm.DB
	node      *CanvasNode
	eventMode nodeResourceEventMode
	limit     int
}

type NodeResourceCleanupResult struct {
	ResourcesDeleted int
	AllDeleted       bool
}

func NewNodeResourceCleaner(tx *gorm.DB, node *CanvasNode) *NodeResourceCleaner {
	return &NodeResourceCleaner{
		tx:    tx,
		node:  node,
		limit: 500,
	}
}

func (c *NodeResourceCleaner) ForAll() *NodeResourceCleaner {
	c.eventMode = nodeResourceEventModeAll
	return c
}

func (c *NodeResourceCleaner) ForUnreferenced() *NodeResourceCleaner {
	c.eventMode = nodeResourceEventModeUnreferenced
	return c
}

func (c *NodeResourceCleaner) WithLimit(limit int) *NodeResourceCleaner {
	c.limit = limit
	return c
}

func (c *NodeResourceCleaner) Run() (NodeResourceCleanupResult, error) {
	if c.node == nil {
		return NodeResourceCleanupResult{}, fmt.Errorf("node is required")
	}
	if c.eventMode == nodeResourceEventModeUnset {
		return NodeResourceCleanupResult{}, fmt.Errorf("event cleanup mode is required")
	}
	if c.limit <= 0 {
		return NodeResourceCleanupResult{}, fmt.Errorf("limit must be positive")
	}

	resourceTypes := []any{
		&CanvasNodeRequest{},
		&CanvasNodeExecutionKV{},
		&CanvasNodeExecution{},
		&CanvasNodeQueueItem{},
		&CanvasEvent{},
	}

	totalDeleted := 0

	for _, model := range resourceTypes {
		if totalDeleted >= c.limit {
			return NodeResourceCleanupResult{ResourcesDeleted: totalDeleted, AllDeleted: false}, nil
		}

		remaining := c.limit - totalDeleted
		var deleted int
		var err error

		if _, isEvent := model.(*CanvasEvent); isEvent {
			deleted, err = c.deleteEventsBatch(remaining)
		} else {
			deleted, err = c.deleteResourceBatch(model, remaining)
		}
		if err != nil {
			return NodeResourceCleanupResult{}, fmt.Errorf("failed to delete resources: %w", err)
		}

		totalDeleted += deleted
		if deleted < remaining {
			continue
		}

		hasMore, err := c.hasRemaining(model)
		if err != nil {
			return NodeResourceCleanupResult{}, err
		}
		if hasMore {
			return NodeResourceCleanupResult{ResourcesDeleted: totalDeleted, AllDeleted: false}, nil
		}
	}

	hasEvents, err := c.hasRemaining(&CanvasEvent{})
	if err != nil {
		return NodeResourceCleanupResult{}, err
	}
	if hasEvents {
		return NodeResourceCleanupResult{ResourcesDeleted: totalDeleted, AllDeleted: false}, nil
	}

	return NodeResourceCleanupResult{ResourcesDeleted: totalDeleted, AllDeleted: true}, nil
}

func (c *NodeResourceCleaner) hasRemaining(model any) (bool, error) {
	var rows []map[string]any
	err := c.tx.Unscoped().Model(model).
		Where("workflow_id = ? AND node_id = ?", c.node.WorkflowID, c.node.NodeID).
		Limit(1).
		Find(&rows).Error
	if err != nil {
		return false, fmt.Errorf("failed to check remaining resources: %w", err)
	}
	return len(rows) > 0, nil
}

func (c *NodeResourceCleaner) deleteResourceBatch(model any, limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}

	// PostgreSQL has no DELETE ... LIMIT, so limit via a SELECT subquery.
	ids := c.tx.Model(model).
		Select("id").
		Where("workflow_id = ? AND node_id = ?", c.node.WorkflowID, c.node.NodeID).
		Limit(limit)

	result := c.tx.Where("id IN (?)", ids).Delete(model)
	if result.Error != nil {
		return 0, result.Error
	}

	return int(result.RowsAffected), nil
}

func (c *NodeResourceCleaner) deleteEventsBatch(limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}

	if c.eventMode == nodeResourceEventModeAll {
		return c.deleteResourceBatch(&CanvasEvent{}, limit)
	}

	ids := c.unreferencedEventsQuery().Limit(limit)
	result := c.tx.Where("id IN (?)", ids).Delete(&CanvasEvent{})
	if result.Error != nil {
		return 0, result.Error
	}

	return int(result.RowsAffected), nil
}

func (c *NodeResourceCleaner) unreferencedEventsQuery() *gorm.DB {
	return c.tx.Model(&CanvasEvent{}).
		Select("id").
		Where("workflow_id = ? AND node_id = ?", c.node.WorkflowID, c.node.NodeID).
		Where(`NOT EXISTS (
			SELECT 1 FROM workflow_node_executions x
			WHERE x.root_event_id = workflow_events.id OR x.event_id = workflow_events.id
		)`).
		Where(`NOT EXISTS (
			SELECT 1 FROM workflow_node_queue_items q
			WHERE q.root_event_id = workflow_events.id OR q.event_id = workflow_events.id
		)`)
}
