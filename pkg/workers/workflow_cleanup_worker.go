package workers

import (
	"context"
	"time"

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
			w.logger.Info("Workflow already being processed - skipping")
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

	if err := tx.Unscoped().Where("workflow_id = ?", workflow.ID).Delete(&models.WorkflowEvent{}).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("workflow_id = ?", workflow.ID).Delete(&models.WorkflowNodeExecution{}).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("workflow_id = ?", workflow.ID).Delete(&models.WorkflowNodeQueueItem{}).Error; err != nil {
		return err
	}

	if err := tx.Unscoped().Where("workflow_id = ?", workflow.ID).Delete(&models.WorkflowNodeExecutionKV{}).Error; err != nil {
		return err
	}

	if err := tx.Unscoped().Where("workflow_id = ?", workflow.ID).Delete(&models.WorkflowNodeRequest{}).Error; err != nil {
		return err
	}

	nodes, err := models.FindWorkflowNodesInTransaction(tx, workflow.ID)
	if err != nil {
		return err
	}

	w.logger.Infof("Found %d nodes to delete for workflow %s", len(nodes), workflow.ID)

	for _, node := range nodes {
		if err := models.DeleteWorkflowNode(tx, node); err != nil {
			return err
		}
	}

	if err := tx.Unscoped().Delete(&workflow).Error; err != nil {
		return err
	}

	w.logger.Infof("Successfully cleaned up workflow %s", workflow.ID)
	return nil
}
