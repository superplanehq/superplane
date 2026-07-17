package workers

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type canvasNodeEventCleanupMode int

const (
	// canvasNodeEventCleanupDeleteAll deletes every event for the node.
	// Used by CanvasCleanupWorker when the whole canvas is being torn down.
	canvasNodeEventCleanupDeleteAll canvasNodeEventCleanupMode = iota
	// canvasNodeEventCleanupDeleteUnreferenced deletes only events that are not
	// still referenced by executions or queue items on other nodes. Remaining
	// events stay for EventRetentionWorker / later cleanup ticks.
	canvasNodeEventCleanupDeleteUnreferenced
)

func deleteCanvasNodeResourcesBatched(
	tx *gorm.DB,
	workflowID uuid.UUID,
	nodeID string,
	maxResources int,
	eventCleanup canvasNodeEventCleanupMode,
) (resourcesDeleted int, allResourcesDeleted bool, err error) {
	resourceTypes := []struct {
		model any
	}{
		{&models.CanvasNodeRequest{}},
		{&models.CanvasNodeExecutionKV{}},
		{&models.CanvasNodeExecution{}},
		{&models.CanvasNodeQueueItem{}},
		{&models.CanvasEvent{}},
	}

	totalDeleted := 0

	for _, resourceType := range resourceTypes {
		if totalDeleted >= maxResources {
			return totalDeleted, false, nil
		}

		remaining := maxResources - totalDeleted
		var deleted int
		var deleteErr error

		if _, isEvent := resourceType.model.(*models.CanvasEvent); isEvent {
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
	if err := tx.Unscoped().Model(&models.CanvasEvent{}).Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Count(&remainingEvents).Error; err != nil {
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
	eventCleanup canvasNodeEventCleanupMode,
) (int, error) {
	if limit <= 0 {
		return 0, nil
	}

	if eventCleanup == canvasNodeEventCleanupDeleteAll {
		return deleteCanvasNodeResourceBatch(tx, &models.CanvasEvent{}, workflowID, nodeID, limit)
	}

	ids := tx.Model(&models.CanvasEvent{}).
		Select("id").
		Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).
		Where(`NOT EXISTS (
			SELECT 1 FROM workflow_node_executions x
			WHERE x.root_event_id = workflow_events.id OR x.event_id = workflow_events.id
		)`).
		Where(`NOT EXISTS (
			SELECT 1 FROM workflow_node_queue_items q
			WHERE q.root_event_id = workflow_events.id OR q.event_id = workflow_events.id
		)`).
		Limit(limit)

	result := tx.Where("id IN (?)", ids).Delete(&models.CanvasEvent{})
	if result.Error != nil {
		return 0, result.Error
	}

	return int(result.RowsAffected), nil
}
