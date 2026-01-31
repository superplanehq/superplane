package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const WorkflowEventCreatedRoutingKey = "workflow-event-created"

type CanvasEventCreatedMessage struct {
	message *pb.CanvasNodeEventMessage
}

func NewCanvasEventCreatedMessage(canvasId string, event *models.CanvasEvent) CanvasEventCreatedMessage {
	return CanvasEventCreatedMessage{
		message: &pb.CanvasNodeEventMessage{
			Id:        event.ID.String(),
			CanvasId:  canvasId,
			NodeId:    event.NodeID,
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m CanvasEventCreatedMessage) Publish() error {
	return Publish(WorkflowExchange, WorkflowEventCreatedRoutingKey, toBytes(m.message))
}
