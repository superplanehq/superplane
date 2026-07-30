package factories

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func CreateWorkOrder(ctx context.Context, organizationID string, req *pb.CreateWorkOrderRequest) (*pb.CreateWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		return nil, factoryErrorToStatus(invalidArgument("title is required"), "failed to create work order")
	}

	tx := database.DB(ctx)
	if _, err := loadFactory(tx, orgID, factoryID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	assigneeIDs, err := parseAssigneeIDs(tx, orgID, req.GetAssigneeIds())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	var order *models.FactoryWorkOrder
	err = tx.Transaction(func(tx *gorm.DB) error {
		var createErr error
		order, createErr = models.CreateFactoryWorkOrder(tx, models.CreateFactoryWorkOrderParams{
			OrganizationID: orgID,
			FactoryID:      factoryID,
			Title:          title,
			Description:    req.GetDescription(),
			AssigneeIDs:    assigneeIDs,
		})
		if createErr != nil {
			return createErr
		}

		content := map[string]any{}
		if userID, ok := authentication.GetUserIdFromMetadata(ctx); ok {
			content["actor_id"] = userID
		}
		if len(assigneeIDs) > 0 {
			content["assignee_ids"] = assigneeIDsToStrings(assigneeIDs)
		}

		_, createErr = models.CreateFactoryWorkOrderEvent(tx, order.ID, workOrderEventTypeCreated, content)
		return createErr
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	return &pb.CreateWorkOrderResponse{
		Order: serializeWorkOrder(order),
	}, nil
}
