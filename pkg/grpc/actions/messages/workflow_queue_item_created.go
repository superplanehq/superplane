package messages

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const WorkflowQueueItemCreatedRoutingKey = "workflow-queue-item-created"

type WorkflowQueueItemCreatedMessage struct {
	message *pb.QueueItemCreated
}

func NewWorkflowQueueItemCreatedMessage(workflowId string, item *models.WorkflowNodeQueueItem) WorkflowQueueItemCreatedMessage {
	return WorkflowQueueItemCreatedMessage{
		message: &pb.QueueItemCreated{
			Id:         item.ID.String(),
			WorkflowId: workflowId,
			NodeId:     item.NodeID,
			Timestamp:  timestamppb.Now(),
		},
	}
}

func (m WorkflowQueueItemCreatedMessage) Publish() error {
	return Publish(WorkflowExchange, WorkflowQueueItemCreatedRoutingKey, toBytes(m.message))
}

func (m WorkflowQueueItemCreatedMessage) PublishWithDelay(delay time.Duration) {
	go func() {
		time.Sleep(delay)
		if err := m.Publish(); err != nil {
			log.Errorf("failed to publish queue item created event: %v", err)
		}
	}()
}
