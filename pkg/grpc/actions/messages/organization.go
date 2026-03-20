package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const OrganizationCreatedRoutingKey = "organization-created"
const OrganizationPlanChangedRoutingKey = "organization-plan-changed"

type OrganizationCreatedMessage struct {
	message *pb.OrganizationCreated
}

type OrganizationPlanChangedMessage struct {
	message *pb.OrganizationPlanChanged
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

func NewOrganizationPlanChangedMessage(
	organizationID string,
	planName string,
	limits *pb.OrganizationLimits,
) OrganizationPlanChangedMessage {
	return OrganizationPlanChangedMessage{
		message: &pb.OrganizationPlanChanged{
			OrganizationId: organizationID,
			PlanName:       planName,
			Limits:         limits,
			Timestamp:      timestamppb.Now(),
		},
	}
}

func (m OrganizationPlanChangedMessage) Publish() error {
	return Publish(CanvasExchange, OrganizationPlanChangedRoutingKey, toBytes(m.message))
}
