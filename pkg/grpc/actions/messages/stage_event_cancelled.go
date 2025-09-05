package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const StageEventCancelledRoutingKey = "stage-event-cancelled"

type StageEventCancelledMessage struct {
	message *pb.StageEventCancelled
}

func NewStageEventCancelledMessage(canvasId string, stageEvent *models.StageEvent) StageEventCancelledMessage {
	return StageEventCancelledMessage{
		message: &pb.StageEventCancelled{
			CanvasId:  canvasId,
			StageId:   stageEvent.StageID.String(),
			EventId:   stageEvent.ID.String(),
			SourceId:  stageEvent.SourceID.String(),
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m StageEventCancelledMessage) Publish() error {
	return Publish(DeliveryHubCanvasExchange, StageEventCancelledRoutingKey, toBytes(m.message))
}