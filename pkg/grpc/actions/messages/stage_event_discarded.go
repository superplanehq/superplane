package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const StageEventDiscardedRoutingKey = "stage-event-discarded"

type StageEventDiscardedMessage struct {
	message *pb.StageEventDiscarded
}

func NewStageEventDiscardedMessage(canvasId string, stageEvent *models.StageEvent) StageEventDiscardedMessage {
	return StageEventDiscardedMessage{
		message: &pb.StageEventDiscarded{
			CanvasId:  canvasId,
			StageId:   stageEvent.StageID.String(),
			EventId:   stageEvent.ID.String(),
			SourceId:  stageEvent.SourceID.String(),
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m StageEventDiscardedMessage) Publish() error {
	return Publish(DeliveryHubCanvasExchange, StageEventDiscardedRoutingKey, toBytes(m.message))
}
