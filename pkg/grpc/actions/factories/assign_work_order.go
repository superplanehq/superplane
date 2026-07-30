package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func AssignWorkOrder(ctx context.Context, organizationID string, req *pb.AssignWorkOrderRequest) (*pb.AssignWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to assign work order")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to assign work order")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to assign work order")
	}

	tx := database.DB(ctx)
	assigneeIDs, err := parseAssigneeIDs(tx, orgID, req.GetAssigneeIds())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to assign work order")
	}

	var order *models.FactoryWorkOrder
	err = tx.Transaction(func(tx *gorm.DB) error {
		var assignErr error
		order, assignErr = models.AssignFactoryWorkOrder(tx, orgID, factoryID, orderID, assigneeIDs)
		if assignErr != nil {
			return assignErr
		}

		content := map[string]any{
			"assignee_ids": assigneeIDsToStrings(assigneeIDs),
		}
		if userID, ok := authentication.GetUserIdFromMetadata(ctx); ok {
			content["actor_id"] = userID
		}

		_, assignErr = models.CreateFactoryWorkOrderEvent(tx, order.ID, workOrderEventTypeAssigned, content)
		return assignErr
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to assign work order")
	}

	return &pb.AssignWorkOrderResponse{
		Order: serializeWorkOrder(order),
	}, nil
}
