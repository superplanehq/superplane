package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func UpdateWorkOrderAssignees(
	ctx context.Context,
	organizationID string,
	req *pb.UpdateWorkOrderAssigneesRequest,
) (*pb.UpdateWorkOrderAssigneesResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	tx := database.DB(ctx)
	assigneeIDs, err := parseAssigneeIDs(tx, orgID, req.GetAssigneeIds())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	order, err := models.UpdateFactoryWorkOrderAssignees(tx, orgID, factoryID, orderID, assigneeIDs)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order assignees")
	}

	return &pb.UpdateWorkOrderAssigneesResponse{
		Order: serialized,
	}, nil
}
