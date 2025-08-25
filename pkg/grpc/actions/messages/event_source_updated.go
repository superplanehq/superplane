package messages

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const EventSourceUpdatedRoutingKey = "event-source-updated"

type EventSourceUpdatedMessage struct {
	message *pb.EventSourceUpdated
}

func NewEventSourceUpdatedMessage(eventSource *models.EventSource, oldResourceID, newResourceID *uuid.UUID) EventSourceUpdatedMessage {
	oldResID := ""
	newResID := ""
	
	if oldResourceID != nil {
		oldResID = oldResourceID.String()
	}
	if newResourceID != nil {
		newResID = newResourceID.String()
	}
	
	return EventSourceUpdatedMessage{
		message: &pb.EventSourceUpdated{
			CanvasId:      eventSource.CanvasID.String(),
			SourceId:      eventSource.ID.String(),
			Timestamp:     timestamppb.Now(),
			OldResourceId: oldResID,
			NewResourceId: newResID,
		},
	}
}

func (m EventSourceUpdatedMessage) Publish() error {
	log.Infof("publishing event source updated message")
	return Publish(DeliveryHubCanvasExchange, EventSourceUpdatedRoutingKey, toBytes(m.message))
}