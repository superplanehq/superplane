package messages

import (
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const AlertCreatedRoutingKey = "alert-created"

type AlertCreatedMessage struct {
	message *pb.AlertCreated
}

func NewAlertCreatedMessage(alert *models.Alert) AlertCreatedMessage {
	return AlertCreatedMessage{
		message: &pb.AlertCreated{
			CanvasId:  alert.CanvasID.String(),
			AlertId:   alert.ID.String(),
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m AlertCreatedMessage) Publish() error {
	log.Infof("publishing alert created message")
	return Publish(DeliveryHubCanvasExchange, AlertCreatedRoutingKey, toBytes(m.message))
}
