package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const WorkflowQueueItemRoutingKey = "workflow-queue-item"

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

func (m WorkflowQueueItemMessage) Publish() error {
	return Publish(WorkflowExchange, WorkflowQueueItemRoutingKey, toBytes(m.message))
}
