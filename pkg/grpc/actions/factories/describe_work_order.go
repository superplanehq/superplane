package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func DescribeWorkOrder(ctx context.Context, organizationID string, req *pb.DescribeWorkOrderRequest) (*pb.DescribeWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe work order")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe work order")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe work order")
	}

	order, err := models.FindFactoryWorkOrder(database.DB(ctx), orgID, factoryID, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe work order")
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe work order")
	}

	return &pb.DescribeWorkOrderResponse{
		Order: serialized,
	}, nil
}
