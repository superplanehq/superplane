package messages

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const WorkflowQueueItemDeletedRoutingKey = "workflow-queue-item-deleted"

type WorkflowQueueItemDeletedMessage struct {
	message *pb.QueueItemDeleted
}

func NewWorkflowQueueItemDeletedMessage(workflowId string, item *models.WorkflowNodeQueueItem) WorkflowQueueItemDeletedMessage {
	return WorkflowQueueItemDeletedMessage{
		message: &pb.QueueItemDeleted{
			Id:         item.ID.String(),
			WorkflowId: workflowId,
			NodeId:     item.NodeID,
			Timestamp:  timestamppb.Now(),
		},
	}
}

func (m WorkflowQueueItemDeletedMessage) Publish() error {
	return Publish(WorkflowExchange, WorkflowQueueItemDeletedRoutingKey, toBytes(m.message))
}

func (m WorkflowQueueItemDeletedMessage) PublishWithDelay(delay time.Duration) {
	go func() {
		time.Sleep(delay)
		if err := m.Publish(); err != nil {
			log.Errorf("failed to publish queue item deleted event: %v", err)
		}
	}()
}
