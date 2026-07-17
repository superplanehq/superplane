package models

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CanvasNodeEventCleanupMode int

const (
	// CanvasNodeEventCleanupDeleteAll deletes every event for the node.
	// Used when the whole canvas is being torn down.
	CanvasNodeEventCleanupDeleteAll CanvasNodeEventCleanupMode = iota
	// CanvasNodeEventCleanupDeleteUnreferenced deletes only events that are not
	// still referenced by executions or queue items on other nodes.
	CanvasNodeEventCleanupDeleteUnreferenced
)

func HardDeleteCanvasNode(tx *gorm.DB, workflowID uuid.UUID, nodeID string) error {
	return tx.Unscoped().
		Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).
		Delete(&CanvasNode{}).
		Error
}

func DeleteCanvasNodeResourcesBatched(
	tx *gorm.DB,
	workflowID uuid.UUID,
	nodeID string,
	maxResources int,
	eventCleanup CanvasNodeEventCleanupMode,
) (resourcesDeleted int, allResourcesDeleted bool, err error) {
	resourceTypes := []struct {
		model any
	}{
		{&CanvasNodeRequest{}},
		{&CanvasNodeExecutionKV{}},
		{&CanvasNodeExecution{}},
		{&CanvasNodeQueueItem{}},
		{&CanvasEvent{}},
	}

	totalDeleted := 0

	for _, resourceType := range resourceTypes {
		if totalDeleted >= maxResources {
			return totalDeleted, false, nil
		}

		remaining := maxResources - totalDeleted
		var deleted int
		var deleteErr error

		if _, isEvent := resourceType.model.(*CanvasEvent); isEvent {
			deleted, deleteErr = deleteCanvasNodeEventsBatch(tx, workflowID, nodeID, remaining, eventCleanup)
		} else {
			deleted, deleteErr = deleteCanvasNodeResourceBatch(tx, resourceType.model, workflowID, nodeID, remaining)
		}
		if deleteErr != nil {
			return totalDeleted, false, fmt.Errorf("failed to delete resources: %w", deleteErr)
		}

		totalDeleted += deleted
		if deleted < remaining {
			continue
		}

		var count int64
		if err := tx.Unscoped().Model(resourceType.model).Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Count(&count).Error; err != nil {
			return totalDeleted, false, fmt.Errorf("failed to count remaining resources: %w", err)
		}

		if count > 0 {
			return totalDeleted, false, nil
		}
	}

	var remainingEvents int64
	if err := tx.Unscoped().Model(&CanvasEvent{}).Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Count(&remainingEvents).Error; err != nil {
		return totalDeleted, false, fmt.Errorf("failed to count remaining workflow_events: %w", err)
	}
	if remainingEvents > 0 {
		return totalDeleted, false, nil
	}

	return totalDeleted, true, nil
}

func deleteCanvasNodeResourceBatch(
	tx *gorm.DB,
	model any,
	workflowID uuid.UUID,
	nodeID string,
	limit int,
) (int, error) {
	if limit <= 0 {
		return 0, nil
	}

	// PostgreSQL has no DELETE ... LIMIT, so limit via a SELECT subquery.
	ids := tx.Model(model).
		Select("id").
		Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).
		Limit(limit)

	result := tx.Where("id IN (?)", ids).Delete(model)
	if result.Error != nil {
		return 0, result.Error
	}

	return int(result.RowsAffected), nil
}

func deleteCanvasNodeEventsBatch(
	tx *gorm.DB,
	workflowID uuid.UUID,
	nodeID string,
	limit int,
	eventCleanup CanvasNodeEventCleanupMode,
) (int, error) {
	if limit <= 0 {
		return 0, nil
	}

	if eventCleanup == CanvasNodeEventCleanupDeleteAll {
		return deleteCanvasNodeResourceBatch(tx, &CanvasEvent{}, workflowID, nodeID, limit)
	}

	ids := unreferencedCanvasNodeEventsQuery(tx, workflowID, nodeID).Limit(limit)
	result := tx.Where("id IN (?)", ids).Delete(&CanvasEvent{})
	if result.Error != nil {
		return 0, result.Error
	}

	return int(result.RowsAffected), nil
}

func unreferencedCanvasNodeEventsQuery(tx *gorm.DB, workflowID uuid.UUID, nodeID string) *gorm.DB {
	return tx.Model(&CanvasEvent{}).
		Select("id").
		Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).
		Where(`NOT EXISTS (
			SELECT 1 FROM workflow_node_executions x
			WHERE x.root_event_id = workflow_events.id OR x.event_id = workflow_events.id
		)`).
		Where(`NOT EXISTS (
			SELECT 1 FROM workflow_node_queue_items q
			WHERE q.root_event_id = workflow_events.id OR q.event_id = workflow_events.id
		)`)
}
