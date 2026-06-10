package workers

import (
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
)

func PublishOnErrorDispatch(dispatch *models.OnErrorDispatch) {
	if dispatch == nil {
		return
	}

	messages.NewCanvasQueueItemMessage(
		dispatch.QueueItem.WorkflowID.String(),
		dispatch.QueueItem.ID.String(),
		dispatch.QueueItem.NodeID,
	).Publish(false)

	messages.PublishCanvasEventCreatedMessage(&dispatch.Event)
}
