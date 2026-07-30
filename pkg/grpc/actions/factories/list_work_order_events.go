package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func ListWorkOrderEvents(ctx context.Context, organizationID string, req *pb.ListWorkOrderEventsRequest) (*pb.ListWorkOrderEventsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	tx := database.DB(ctx)
	if _, err := models.FindFactoryWorkOrder(tx, orgID, factoryID, orderID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	events, err := models.ListFactoryWorkOrderEvents(tx, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	protoEvents, err := serializeWorkOrderEvents(events)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work order events")
	}

	return &pb.ListWorkOrderEventsResponse{
		Events: protoEvents,
	}, nil
}
