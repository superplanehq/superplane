package workers

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func deleteCanvasNodeResourcesBatched(
	tx *gorm.DB,
	workflowID uuid.UUID,
	nodeID string,
	maxResources int,
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
	allDeleted := true

	for _, resourceType := range resourceTypes {
		if totalDeleted >= maxResources {
			allDeleted = false
			break
		}

		remaining := maxResources - totalDeleted
		deleted, err := deleteCanvasNodeResourceBatch(tx, resourceType.tableName, workflowID, nodeID, remaining)
		if err != nil {
			return totalDeleted, false, fmt.Errorf("failed to delete %s: %w", resourceType.tableName, err)
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
			allDeleted = false
			break
		}
	}

	return totalDeleted, allDeleted, nil
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
