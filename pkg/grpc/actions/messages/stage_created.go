package messages

import (
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const StageCreatedRoutingKey = "stage-created"

type StageCreatedMessage struct {
	message *pb.StageCreated
}

func NewStageCreatedMessage(stage *models.Stage) StageCreatedMessage {
	return StageCreatedMessage{
		message: &pb.StageCreated{
			CanvasId:  stage.CanvasID.String(),
			StageId:   stage.ID.String(),
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m StageCreatedMessage) Publish() error {
	log.Infof("publishing stage created message")
	return Publish(DeliveryHubCanvasExchange, StageCreatedRoutingKey, toBytes(m.message))
}
