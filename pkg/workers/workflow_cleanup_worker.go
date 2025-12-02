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
	semaphore *semaphore.Weighted
	logger    *log.Entry
}

func NewWorkflowCleanupWorker() *WorkflowCleanupWorker {
	return &WorkflowCleanupWorker{
		semaphore: semaphore.NewWeighted(25),
		logger:    log.WithFields(log.Fields{"worker": "WorkflowCleanupWorker"}),
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

	for _, node := range nodes {
		if err := w.deleteNodeResources(tx, workflow.ID, node.NodeID); err != nil {
			return fmt.Errorf("failed to delete resources for node %s: %w", node.NodeID, err)
		}

		if err := tx.Unscoped().Where("workflow_id = ? AND node_id = ?", workflow.ID, node.NodeID).Delete(&models.WorkflowNode{}).Error; err != nil {
			return fmt.Errorf("failed to delete workflow node %s: %w", node.NodeID, err)
		}
	}

	if err := tx.Unscoped().Delete(&workflow).Error; err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	w.logger.Infof("Successfully cleaned up workflow %s with %d nodes", workflow.ID, len(nodes))
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
