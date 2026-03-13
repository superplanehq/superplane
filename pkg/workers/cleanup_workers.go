package workers

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// deleteNodeResourcesBatched deletes resources associated with a workflow node
// in batches, respecting a maximum resource limit per call.
func deleteNodeResourcesBatched(tx *gorm.DB, workflowID uuid.UUID, nodeID string, maxResources int) (resourcesDeleted int, allResourcesDeleted bool, err error) {
	resourceTypes := []struct {
		model     any
		tableName string
	}{
		{&models.CanvasNodeRequest{}, "canvas_node_requests"},
		{&models.CanvasNodeExecutionKV{}, "canvas_node_execution_kvs"},
		{&models.CanvasNodeExecution{}, "canvas_node_executions"},
		{&models.CanvasNodeQueueItem{}, "canvas_node_queue_items"},
		{&models.CanvasEvent{}, "canvas_events"},
	}

	totalDeleted := 0
	allDeleted := true

	for _, resourceType := range resourceTypes {
		if totalDeleted >= maxResources {
			allDeleted = false
			break
		}

		remaining := maxResources - totalDeleted

		// Delete in batches with LIMIT
		result := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Limit(remaining).Delete(resourceType.model)
		if result.Error != nil {
			return totalDeleted, false, fmt.Errorf("failed to delete %s: %w", resourceType.tableName, result.Error)
		}

		deleted := int(result.RowsAffected)
		totalDeleted += deleted

		if deleted != remaining {
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
