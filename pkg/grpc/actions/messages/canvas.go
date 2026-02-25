package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	WorkflowCanvasUpdatedRoutingKey = "workflow-canvas-updated"
	WorkflowCanvasDeletedRoutingKey = "workflow-canvas-deleted"
)

type CanvasMessage struct {
	message *pb.CanvasMessage
}

func NewCanvasUpdatedMessage(canvasID string) CanvasMessage {
	return CanvasMessage{
		message: &pb.CanvasMessage{
			Id:        canvasID,
			CanvasId:  canvasID,
			Timestamp: timestamppb.Now(),
		},
	}
}

func NewCanvasDeletedMessage(canvasID string) CanvasMessage {
	return CanvasMessage{
		message: &pb.CanvasMessage{
			Id:        canvasID,
			CanvasId:  canvasID,
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m CanvasMessage) Publish(updated bool) error {
	if updated {
		return Publish(WorkflowExchange, WorkflowCanvasUpdatedRoutingKey, toBytes(m.message))
	}

	return Publish(WorkflowExchange, WorkflowCanvasDeletedRoutingKey, toBytes(m.message))
}
