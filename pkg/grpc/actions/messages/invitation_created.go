package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const InvitationCreatedRoutingKey = "invitation-created"

type InvitationCreatedMessage struct {
	message *pb.InvitationCreated
}

func NewInvitationCreatedMessage(invitation *models.OrganizationInvitation) InvitationCreatedMessage {
	return InvitationCreatedMessage{
		message: &pb.InvitationCreated{
			InvitationId: invitation.ID.String(),
			Timestamp:    timestamppb.Now(),
		},
	}
}

func (m InvitationCreatedMessage) Publish() error {
	return Publish(DeliveryHubCanvasExchange, InvitationCreatedRoutingKey, toBytes(m.message))
}