package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/delivery"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const StageEventCreatedRoutingKey = "stage-event-created"

type StageEventCreatedMessage struct {
	message *pb.StageEventCreated
}

func NewStageEventCreatedMessage(canvasId string, eventSource *models.StageEvent) StageEventCreatedMessage {
	return StageEventCreatedMessage{
		message: &pb.StageEventCreated{
			CanvasId:  canvasId,
			StageId:   eventSource.StageID.String(),
			EventId:   eventSource.ID.String(),
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m StageEventCreatedMessage) Publish() error {
	return Publish(DeliveryHubCanvasExchange, StageEventCreatedRoutingKey, toBytes(m.message))
}
