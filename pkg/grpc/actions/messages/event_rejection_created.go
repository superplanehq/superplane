package messages

import (
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const EventRejectionCreatedRoutingKey = "event-rejection-created"

type EventRejectionCreatedMessage struct {
	message *pb.EventRejectionCreated
}

func NewEventRejectionCreatedMessage(rejection *models.EventRejection) EventRejectionCreatedMessage {
	return EventRejectionCreatedMessage{
		message: &pb.EventRejectionCreated{
			RejectionId: rejection.ID.String(),
			Timestamp:   timestamppb.Now(),
		},
	}
}

func (m EventRejectionCreatedMessage) Publish() error {
	log.Infof("publishing event rejection created message")
	return Publish(DeliveryHubCanvasExchange, EventRejectionCreatedRoutingKey, toBytes(m.message))
}
