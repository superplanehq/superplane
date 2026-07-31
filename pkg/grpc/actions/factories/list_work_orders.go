package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func ListWorkOrders(ctx context.Context, organizationID string, req *pb.ListWorkOrdersRequest) (*pb.ListWorkOrdersResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work orders")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work orders")
	}

	tx := database.DB(ctx)
	if _, err := loadFactory(tx, orgID, factoryID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work orders")
	}

	orders, err := models.ListFactoryWorkOrders(tx, orgID, factoryID, listWorkOrderFilters(req))
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work orders")
	}

	serialized, err := loadAndSerializeWorkOrders(ctx, orders)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list work orders")
	}

	return &pb.ListWorkOrdersResponse{
		Orders: serialized,
	}, nil
}
