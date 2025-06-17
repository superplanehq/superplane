package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const StageEventCreatedRoutingKey = "stage-event-created"

type StageEventCreatedMessage struct {
	message *pb.StageEventCreated
}

func NewStageEventCreatedMessage(canvasId string, stageEvent *models.StageEvent) StageEventCreatedMessage {
	return StageEventCreatedMessage{
		message: &pb.StageEventCreated{
			CanvasId:  canvasId,
			StageId:   stageEvent.StageID.String(),
			EventId:   stageEvent.ID.String(),
			SourceId:  stageEvent.SourceID.String(),
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m StageEventCreatedMessage) Publish() error {
	return Publish(DeliveryHubCanvasExchange, StageEventCreatedRoutingKey, toBytes(m.message))
}
