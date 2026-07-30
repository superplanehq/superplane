package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
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

	tx := database.DB(ctx)
	var order *models.FactoryWorkOrder
	err = tx.Transaction(func(tx *gorm.DB) error {
		var closeErr error
		order, closeErr = models.CloseFactoryWorkOrder(tx, orgID, factoryID, orderID, result)
		if closeErr != nil {
			return closeErr
		}

		content := map[string]any{
			"result": result,
		}
		if userID, ok := authentication.GetUserIdFromMetadata(ctx); ok {
			content["actor_id"] = userID
		}

		_, closeErr = models.CreateFactoryWorkOrderEvent(tx, order.ID, workOrderEventTypeClosed, content)
		return closeErr
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	return &pb.CloseWorkOrderResponse{
		Order: serializeWorkOrder(order),
	}, nil
}
