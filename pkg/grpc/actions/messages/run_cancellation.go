package messages

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
)

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
}
