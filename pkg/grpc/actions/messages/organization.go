package messages

import (
	organizationpb "github.com/superplanehq/superplane/pkg/protos/organizations"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const OrganizationCreatedRoutingKey = "organization-created"
const OrganizationPlanChangedRoutingKey = "organization-plan-changed"

type OrganizationCreatedMessage struct {
	message *organizationpb.OrganizationCreated
}

type OrganizationPlanChangedMessage struct {
	message *usagepb.OrganizationPlanChanged
}

func NewOrganizationCreatedMessage(organizationID string) OrganizationCreatedMessage {
	return OrganizationCreatedMessage{
		message: &organizationpb.OrganizationCreated{
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
	limits *usagepb.OrganizationLimits,
) OrganizationPlanChangedMessage {
	return OrganizationPlanChangedMessage{
		message: &usagepb.OrganizationPlanChanged{
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
