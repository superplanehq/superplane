package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func CloseWorkOrder(ctx context.Context, organizationID string, req *pb.CloseWorkOrderRequest) (*pb.CloseWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	result, err := closeWorkOrderResult(req.GetResult())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	logger := logging.ForFactory(*factory)
	order, err := factory.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	logger = logging.WithWorkOrder(logger, *order)
	order, err = order.Close(db, result)
	if err != nil {
		logger.WithError(err).Error("close work order failed")
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	return &pb.CloseWorkOrderResponse{
		Order: serialized,
	}, nil
}
