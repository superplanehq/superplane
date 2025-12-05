package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/telemetry"
)

type WorkflowCleanupWorker struct {
	semaphore           *semaphore.Weighted
	logger              *log.Entry
	maxResourcesPerTick int
}

func NewWorkflowCleanupWorker() *WorkflowCleanupWorker {
	return &WorkflowCleanupWorker{
		semaphore:           semaphore.NewWeighted(25),
		logger:              log.WithFields(log.Fields{"worker": "WorkflowCleanupWorker"}),
		maxResourcesPerTick: 500,
	}
}

func (w *WorkflowCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()
			workflows, err := models.ListDeletedWorkflows()
			if err != nil {
				w.logger.Errorf("Error finding deleted workflows: %v", err)
				continue
			}

			telemetry.RecordWorkflowCleanupWorkerWorkflowsCount(context.Background(), len(workflows))

			for _, workflow := range workflows {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(workflow models.Workflow) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessWorkflow(workflow); err != nil {
						w.logger.Errorf("Error processing workflow %s: %v", workflow.ID, err)
					}
				}(workflow)
			}

			telemetry.RecordWorkflowCleanupWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *WorkflowCleanupWorker) LockAndProcessWorkflow(workflow models.Workflow) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedWorkflow, err := models.LockWorkflow(tx, workflow.ID)
		if err != nil {
			w.logger.Infof("Workflow %s already being processed - skipping", workflow.ID)
			return nil
		}

		w.logger.Infof("Processing deleted workflow %s", lockedWorkflow.ID)
		return w.processWorkflow(tx, *lockedWorkflow)
	})
}

func (w *WorkflowCleanupWorker) processWorkflow(tx *gorm.DB, workflow models.Workflow) error {
	if !workflow.DeletedAt.Valid {
		w.logger.Infof("Skipping non-deleted workflow %s", workflow.ID)
		return nil
	}

	var nodes []models.WorkflowNode
	err := tx.Unscoped().Where("workflow_id = ?", workflow.ID).Find(&nodes).Error
	if err != nil {
		return fmt.Errorf("failed to find workflow nodes: %w", err)
	}

	totalResourcesDeleted := 0
	nodesProcessed := 0

	for _, node := range nodes {
		if totalResourcesDeleted >= w.maxResourcesPerTick {
			w.logger.Infof("Reached max resources per tick (%d), stopping for this cycle", w.maxResourcesPerTick)
			break
		}

		resourcesDeleted, allResourcesDeleted, err := w.deleteNodeResourcesBatched(tx, workflow.ID, node.NodeID, w.maxResourcesPerTick-totalResourcesDeleted)
		if err != nil {
			return fmt.Errorf("failed to delete resources for node %s: %w", node.NodeID, err)
		}

		totalResourcesDeleted += resourcesDeleted

		if !allResourcesDeleted {
			w.logger.Infof("Partially cleaned node %s from workflow %s (deleted %d resources, more remain)", node.NodeID, workflow.ID, resourcesDeleted)
			nodesProcessed++

			continue
		}

		if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", workflow.ID, node.NodeID).Delete(&models.WorkflowNode{}).Error; err != nil {
			return fmt.Errorf("failed to delete workflow node %s: %w", node.NodeID, err)
		}

		w.logger.Infof("Deleted node %s from workflow %s (deleted %d resources)", node.NodeID, workflow.ID, resourcesDeleted)
		nodesProcessed++
	}

	//
	// Check if all nodes are gone, then delete the workflow
	//
	var remainingNodesCount int64
	err = tx.Unscoped().Model(&models.WorkflowNode{}).Where("workflow_id = ?", workflow.ID).Count(&remainingNodesCount).Error
	if err != nil {
		return fmt.Errorf("failed to check remaining workflow nodes: %w", err)
	}

	if remainingNodesCount > 0 {
		w.logger.Infof("Processed %d nodes from workflow %s (deleted %d resources, %d nodes remaining)", nodesProcessed, workflow.ID, totalResourcesDeleted, remainingNodesCount)
		return nil
	}

	w.logger.Infof("Processed %d nodes from workflow %s (deleted %d resources, %d nodes remaining)", nodesProcessed, workflow.ID, totalResourcesDeleted, remainingNodesCount)
	if err := tx.Unscoped().Delete(&workflow).Error; err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	w.logger.Infof("Successfully cleaned up workflow %s (deleted %d resources total)", workflow.ID, totalResourcesDeleted)
	return nil
}

func (w *WorkflowCleanupWorker) deleteNodeResources(tx *gorm.DB, workflowID uuid.UUID, nodeID string) error {
	if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Delete(&models.WorkflowNodeRequest{}).Error; err != nil {
		return fmt.Errorf("failed to delete node requests: %w", err)
	}

	if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Delete(&models.WorkflowNodeExecutionKV{}).Error; err != nil {
		return fmt.Errorf("failed to delete node execution KVs: %w", err)
	}

	if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Delete(&models.WorkflowNodeExecution{}).Error; err != nil {
		return fmt.Errorf("failed to delete node executions: %w", err)
	}

	if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Delete(&models.WorkflowNodeQueueItem{}).Error; err != nil {
		return fmt.Errorf("failed to delete node queue items: %w", err)
	}

	if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Delete(&models.WorkflowEvent{}).Error; err != nil {
		return fmt.Errorf("failed to delete workflow events: %w", err)
	}

	return nil
}

func (w *WorkflowCleanupWorker) deleteNodeResourcesBatched(tx *gorm.DB, workflowID uuid.UUID, nodeID string, maxResources int) (resourcesDeleted int, allResourcesDeleted bool, err error) {
	resourceTypes := []struct {
		model     interface{}
		tableName string
	}{
		{&models.WorkflowNodeRequest{}, "workflow_node_requests"},
		{&models.WorkflowNodeExecutionKV{}, "workflow_node_execution_kvs"},
		{&models.WorkflowNodeExecution{}, "workflow_node_executions"},
		{&models.WorkflowNodeQueueItem{}, "workflow_node_queue_items"},
		{&models.WorkflowEvent{}, "workflow_events"},
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
