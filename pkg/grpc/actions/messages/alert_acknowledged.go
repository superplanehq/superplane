package messages

import (
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const AlertAcknowledgedRoutingKey = "alert-acknowledged"

type AlertAcknowledgedMessage struct {
	message *pb.AlertAcknowledged
}

func NewAlertAcknowledgedMessage(alert *models.Alert) AlertAcknowledgedMessage {
	return AlertAcknowledgedMessage{
		message: &pb.AlertAcknowledged{
			CanvasId:  alert.CanvasID.String(),
			AlertId:   alert.ID.String(),
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m AlertAcknowledgedMessage) Publish() error {
	log.Infof("publishing alert created message")
	return Publish(DeliveryHubCanvasExchange, AlertAcknowledgedRoutingKey, toBytes(m.message))
}
