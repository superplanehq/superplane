package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func UpdateWorkOrderStatus(
	ctx context.Context,
	organizationID string,
	req *pb.UpdateWorkOrderStatusRequest,
) (*pb.UpdateWorkOrderStatusResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order status")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order status")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order status")
	}

	toState, ok := workOrderStateFromProto(req.GetState())
	if !ok {
		return nil, factoryErrorToStatus(invalidArgument("state is required"), "failed to update work order status")
	}

	result := ""
	if toState == models.FactoryWorkOrderStateClosed {
		mapped, ok := workOrderResultFromProto(req.GetResult())
		if !ok {
			return nil, factoryErrorToStatus(invalidArgument("closing a work order requires a result"), "failed to update work order status")
		}
		result = mapped
	}

	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	actor, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to update work order status")
	}

	db := database.DB(ctx)
	var order *models.FactoryWorkOrder
	err = db.Transaction(func(tx *gorm.DB) error {
		factory, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}

		order, err = factory.FindWorkOrder(tx, orderID)
		if err != nil {
			return err
		}

		return order.UpdateStatus(tx, models.FactoryWorkOrderStatusUpdate{
			ToState: toState,
			Result:  result,
			Actor:   &actor,
		})
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order status")
	}

	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order status")
	}

	order, err = factory.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order status")
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order status")
	}

	return &pb.UpdateWorkOrderStatusResponse{Order: serialized}, nil
}
