package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const OrganizationCreatedRoutingKey = "organization-created"

type OrganizationCreatedMessage struct {
	message *pb.OrganizationCreated
}

func NewOrganizationCreatedMessage(organizationID string) OrganizationCreatedMessage {
	return OrganizationCreatedMessage{
		message: &pb.OrganizationCreated{
			OrganizationId: organizationID,
			Timestamp:      timestamppb.Now(),
		},
	}
}

func (m OrganizationCreatedMessage) Publish() error {
	return Publish(CanvasExchange, OrganizationCreatedRoutingKey, toBytes(m.message))
}
