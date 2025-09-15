package messages

import (
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const EventCreatedRoutingKey = "event-created"

type EventCreatedMessage struct {
	message *pb.EventCreated
}

func NewEventCreatedMessage(canvasId string, event *models.Event) EventCreatedMessage {
	return EventCreatedMessage{
		message: &pb.EventCreated{
			CanvasId:   canvasId,
			SourceId:   event.SourceID.String(),
			EventId:    event.ID.String(),
			SourceType: actions.EventSourceTypeToProto(event.SourceType),
			Timestamp:  timestamppb.Now(),
		},
	}
}

func (m EventCreatedMessage) Publish() error {
	return Publish(DeliveryHubCanvasExchange, EventCreatedRoutingKey, toBytes(m.message))
}
