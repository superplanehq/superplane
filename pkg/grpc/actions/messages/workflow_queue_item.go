package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	WorkflowQueueItemCreatedRoutingKey  = "workflow-queue-item-created"
	WorkflowQueueItemConsumedRoutingKey = "workflow-queue-item-consumed"
)

type WorkflowQueueItemMessage struct {
	message *pb.WorkflowNodeQueueItemMessage
}

func NewWorkflowQueueItemMessage(workflowId string, queueItemID, nodeID string) WorkflowQueueItemMessage {
	return WorkflowQueueItemMessage{
		message: &pb.WorkflowNodeQueueItemMessage{
			Id:         queueItemID,
			WorkflowId: workflowId,
			NodeId:     nodeID,
			Timestamp:  timestamppb.Now(),
		},
	}
}

func (m WorkflowQueueItemMessage) Publish(consumed bool) error {
	if consumed {
		return Publish(WorkflowExchange, WorkflowQueueItemConsumedRoutingKey, toBytes(m.message))
	}
	return Publish(WorkflowExchange, WorkflowQueueItemCreatedRoutingKey, toBytes(m.message))
}
