package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EventSourceCreatedMessage struct {
	message *pb.EventSourceCreated
}

const EventSourceCreatedRoutingKey = "event-source-created"

func NewEventSourceCreatedMessage(eventSource *models.EventSource) EventSourceCreatedMessage {
	return EventSourceCreatedMessage{
		message: &pb.EventSourceCreated{
			CanvasId:  eventSource.CanvasID.String(),
			SourceId:  eventSource.ID.String(),
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m EventSourceCreatedMessage) Publish() error {
	return Publish(DeliveryHubCanvasExchange, EventSourceCreatedRoutingKey, toBytes(m.message))
}
