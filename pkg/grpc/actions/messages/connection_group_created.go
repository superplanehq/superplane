package messages

import (
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ConnectionGroupCreatedRoutingKey = "connection-group-created"

type ConnectionGroupCreatedMessage struct {
	message *pb.ConnectionGroupCreated
}

func NewConnectionGroupCreatedMessage(connectionGroup *models.ConnectionGroup) ConnectionGroupCreatedMessage {
	return ConnectionGroupCreatedMessage{
		message: &pb.ConnectionGroupCreated{
			CanvasId:          connectionGroup.CanvasID.String(),
			ConnectionGroupId: connectionGroup.ID.String(),
			Timestamp:         timestamppb.Now(),
		},
	}
}

func (m ConnectionGroupCreatedMessage) Publish() error {
	log.Infof("publishing connection group created message")
	return Publish(DeliveryHubCanvasExchange, ConnectionGroupCreatedRoutingKey, toBytes(m.message))
}
