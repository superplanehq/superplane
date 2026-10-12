package messages

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// PublishWorkOrderRunCancellations wakes terminators after a task close commits.
func PublishWorkOrderRunCancellations(db *gorm.DB, order *models.FactoryWorkOrder) {
	runs, err := order.ListRuns(db)
	if err != nil {
		log.WithError(err).Error("failed to load cancelled task runs")
		return
	}
	for i := range runs {
		run := &runs[i]
		if run.State != models.CanvasRunStateCancelling {
			continue
		}
		executions, err := run.ListExecutionsInStates(db, []string{models.CanvasNodeExecutionStateCancelling})
		if err != nil {
			log.WithError(err).Error("failed to load cancelled task executions")
			continue
		}
		for _, execution := range executions {
			if err := PublishCanvasExecutionByID(run.WorkflowID, execution.ID); err != nil {
				log.WithError(err).Error("failed to publish task execution cancelling message")
			}
		}
		if err := NewCanvasRunMessage(run.WorkflowID.String(), run.ID.String()).Publish(); err != nil {
			log.WithError(err).Error("failed to publish task run cancelling message")
		}
	}
}

func PublishRunCancellationDrain(workflowID uuid.UUID, drainResult *models.RunCancellationDrainResult) {
	if drainResult == nil {
		return
	}

	for _, executionID := range drainResult.RequestedExecutionIDs {
		if err := PublishCanvasExecutionByID(workflowID, executionID); err != nil {
			log.Errorf("failed to publish execution cancelling RabbitMQ message: %v", err)
		}
	}

	for _, queueItem := range drainResult.DeletedQueueItems {
		if err := NewCanvasQueueItemMessage(queueItem).PublishDeleted(); err != nil {
			log.Errorf("failed to publish queue item deleted RabbitMQ message: %v", err)
		}
	}

	for _, event := range drainResult.SupersededEvents {
		if err := PublishEventTerminal(event.WorkflowID, event.RunID, event.ID); err != nil {
			log.Errorf("failed to publish event terminal RabbitMQ message: %v", err)
		}
	}

	for _, child := range drainResult.ChildRuns {
		PublishRunCancellationDrain(child.Run.WorkflowID, child.Drain)
		if child.NewlyCancelling {
			if err := NewCanvasRunMessage(child.Run.WorkflowID.String(), child.Run.ID.String()).Publish(); err != nil {
				log.Errorf("failed to publish child run cancelling RabbitMQ message: %v", err)
			}
		}
	}
}
