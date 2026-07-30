package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func ListAgentAssignmentsForOrder(
	ctx context.Context,
	organizationID string,
	req *pb.ListAgentAssignmentsForOrderRequest,
) (*pb.ListAgentAssignmentsForOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent assignments")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent assignments")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent assignments")
	}

	tx := database.DB(ctx)
	if _, err := models.FindFactoryWorkOrder(tx, orgID, factoryID, orderID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent assignments")
	}

	assignments, err := models.ListFactoryAgentAssignmentsForOrder(tx, orgID, factoryID, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent assignments")
	}

	return &pb.ListAgentAssignmentsForOrderResponse{
		Assignments: serializeAgentAssignments(assignments),
	}, nil
}
