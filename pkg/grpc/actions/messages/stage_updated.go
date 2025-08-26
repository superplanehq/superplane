package messages

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const StageUpdatedRoutingKey = "stage-updated"

type StageUpdatedMessage struct {
	message *pb.StageUpdated
}

func NewStageUpdatedMessage(stage *models.Stage, oldResourceID, newResourceID *uuid.UUID) StageUpdatedMessage {
	oldResID := ""
	newResID := ""
	
	if oldResourceID != nil {
		oldResID = oldResourceID.String()
	}
	if newResourceID != nil {
		newResID = newResourceID.String()
	}
	
	return StageUpdatedMessage{
		message: &pb.StageUpdated{
			CanvasId:      stage.CanvasID.String(),
			StageId:       stage.ID.String(),
			Timestamp:     timestamppb.Now(),
			OldResourceId: oldResID,
			NewResourceId: newResID,
		},
	}
}

func (m StageUpdatedMessage) Publish() error {
	log.Infof("publishing stage updated message")
	return Publish(DeliveryHubCanvasExchange, StageUpdatedRoutingKey, toBytes(m.message))
}