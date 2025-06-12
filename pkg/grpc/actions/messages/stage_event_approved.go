package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const StageEventApprovedRoutingKey = "stage-event-approved"

type StageEventApprovedMessage struct {
	message *pb.StageEventApproved
}

func NewStageEventApprovedMessage(canvasId string, stageEvent *models.StageEvent) StageEventApprovedMessage {
	return StageEventApprovedMessage{
		message: &pb.StageEventApproved{
			CanvasId:  canvasId,
			StageId:   stageEvent.StageID.String(),
			EventId:   stageEvent.ID.String(),
			SourceId:  stageEvent.SourceID.String(),
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m StageEventApprovedMessage) Publish() error {
	return Publish(DeliveryHubCanvasExchange, StageEventApprovedRoutingKey, toBytes(m.message))
}
