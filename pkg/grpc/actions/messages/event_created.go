package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const CanvasEventCreatedRoutingKey = "canvas-event-created"

type CanvasEventCreatedMessage struct {
	message *pb.CanvasNodeEventMessage
}

func NewCanvasEventCreatedMessage(canvasId string, organizationID string, event *models.CanvasEvent) CanvasEventCreatedMessage {
	return CanvasEventCreatedMessage{
		message: &pb.CanvasNodeEventMessage{
			Id:             event.ID.String(),
			CanvasId:       canvasId,
			NodeId:         event.NodeID,
			Timestamp:      timestamppb.Now(),
			OrganizationId: organizationID,
		},
	}
}

func (m CanvasEventCreatedMessage) Publish() error {
	return Publish(CanvasExchange, CanvasEventCreatedRoutingKey, toBytes(m.message))
}

func PublishCanvasEventCreatedMessage(event *models.CanvasEvent) error {
	canvas, err := models.FindCanvasWithoutOrgScope(event.WorkflowID)
	if err != nil {
		return err
	}

	return NewCanvasEventCreatedMessage(
		event.WorkflowID.String(),
		canvas.OrganizationID.String(),
		event,
	).Publish()
}
