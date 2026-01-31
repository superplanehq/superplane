package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	WorkflowQueueItemCreatedRoutingKey  = "workflow-queue-item-created"
	WorkflowQueueItemConsumedRoutingKey = "workflow-queue-item-consumed"
)

type CanvasQueueItemMessage struct {
	message *pb.CanvasNodeQueueItemMessage
}

func NewCanvasQueueItemMessage(canvasId string, queueItemID, nodeID string) CanvasQueueItemMessage {
	return CanvasQueueItemMessage{
		message: &pb.CanvasNodeQueueItemMessage{
			Id:        queueItemID,
			CanvasId:  canvasId,
			NodeId:    nodeID,
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m CanvasQueueItemMessage) Publish(consumed bool) error {
	if consumed {
		return Publish(WorkflowExchange, WorkflowQueueItemConsumedRoutingKey, toBytes(m.message))
	}
	return Publish(WorkflowExchange, WorkflowQueueItemCreatedRoutingKey, toBytes(m.message))
}
