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
		model     any
		tableName string
	}{
		{&models.CanvasNodeRequest{}, "workflow_node_requests"},
		{&models.CanvasNodeExecutionKV{}, "workflow_node_execution_kvs"},
		{&models.CanvasNodeExecution{}, "workflow_node_executions"},
		{&models.CanvasNodeQueueItem{}, "workflow_node_queue_items"},
		{&models.CanvasEvent{}, "workflow_events"},
	}

	totalDeleted := 0

	for _, resourceType := range resourceTypes {
		if totalDeleted >= maxResources {
			return totalDeleted, false, nil
		}

		remaining := maxResources - totalDeleted
		var deleted int
		var deleteErr error

		if resourceType.tableName == "workflow_events" {
			deleted, deleteErr = deleteCanvasNodeEventsBatch(tx, workflowID, nodeID, remaining, eventCleanup)
		} else {
			deleted, deleteErr = deleteCanvasNodeResourceBatch(tx, resourceType.tableName, workflowID, nodeID, remaining)
		}
		if deleteErr != nil {
			return totalDeleted, false, fmt.Errorf("failed to delete %s: %w", resourceType.tableName, deleteErr)
		}

		totalDeleted += deleted
		if deleted < remaining {
			continue
		}

		var count int64
		if err := tx.Unscoped().Model(resourceType.model).Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Count(&count).Error; err != nil {
			return totalDeleted, false, fmt.Errorf("failed to count remaining %s: %w", resourceType.tableName, err)
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
	tableName string,
	workflowID uuid.UUID,
	nodeID string,
	limit int,
) (int, error) {
	if limit <= 0 {
		return 0, nil
	}

	// PostgreSQL does not support LIMIT on DELETE, so delete via a subquery.
	result := tx.Exec(
		fmt.Sprintf(
			`DELETE FROM %s WHERE id IN (
				SELECT id FROM %s
				WHERE workflow_id = ? AND node_id = ?
				LIMIT ?
			)`,
			tableName,
			tableName,
		),
		workflowID,
		nodeID,
		limit,
	)
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
		return deleteCanvasNodeResourceBatch(tx, "workflow_events", workflowID, nodeID, limit)
	}

	result := tx.Exec(
		`DELETE FROM workflow_events WHERE id IN (
			SELECT e.id FROM workflow_events e
			WHERE e.workflow_id = ? AND e.node_id = ?
			AND NOT EXISTS (
				SELECT 1 FROM workflow_node_executions x
				WHERE x.root_event_id = e.id OR x.event_id = e.id
			)
			AND NOT EXISTS (
				SELECT 1 FROM workflow_node_queue_items q
				WHERE q.root_event_id = e.id OR q.event_id = e.id
			)
			LIMIT ?
		)`,
		workflowID,
		nodeID,
		limit,
	)
	if result.Error != nil {
		return 0, result.Error
	}

	return int(result.RowsAffected), nil
}
