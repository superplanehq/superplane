package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	CanvasQueueItemCreatedRoutingKey  = "canvas-queue-item-created"
	CanvasQueueItemConsumedRoutingKey = "canvas-queue-item-consumed"
	CanvasQueueItemDeletedRoutingKey  = "canvas-queue-item-deleted"
)

type CanvasQueueItemMessage struct {
	message *pb.CanvasNodeQueueItemMessage
}

func NewCanvasQueueItemMessage(queueItem models.CanvasNodeQueueItem) CanvasQueueItemMessage {
	return CanvasQueueItemMessage{
		message: &pb.CanvasNodeQueueItemMessage{
			Id:        queueItem.ID.String(),
			CanvasId:  queueItem.WorkflowID.String(),
			NodeId:    queueItem.NodeID,
			RunId:     queueItem.RunID.String(),
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m CanvasQueueItemMessage) PublishConsumed() error {
	return Publish(CanvasExchange, CanvasQueueItemConsumedRoutingKey, toBytes(m.message))
}

func (m CanvasQueueItemMessage) PublishCreated() error {
	return Publish(CanvasExchange, CanvasQueueItemCreatedRoutingKey, toBytes(m.message))
}

func (m CanvasQueueItemMessage) PublishDeleted() error {
	return Publish(CanvasExchange, CanvasQueueItemDeletedRoutingKey, toBytes(m.message))
}
