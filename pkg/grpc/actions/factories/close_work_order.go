package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func CloseWorkOrder(ctx context.Context, organizationID string, req *pb.CloseWorkOrderRequest) (*pb.CloseWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	closedBy, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to create work order")
	}

	result, err := closeWorkOrderResult(req.GetResult())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	order, err := findWorkOrder(db, factory, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	order, err = closeWorkOrderAsUser(db, orgID, factory, order, result, closedBy)
	if err != nil {
		logging.WithWorkOrder(logging.ForFactory(*factory), *order).WithError(err).Error("close work order failed")
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	return &pb.CloseWorkOrderResponse{
		Order: serialized,
	}, nil
}

func closeWorkOrderAsUser(
	db *gorm.DB,
	orgID uuid.UUID,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	result string,
	closedBy uuid.UUID,
) (*models.FactoryWorkOrder, error) {
	logger := logging.WithWorkOrder(logging.ForFactory(*factory), *order)
	fromState := order.State
	wasClosed := order.IsClosed()
	closed, err := order.Close(db, result, &closedBy)
	if err != nil {
		return nil, err
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		closed.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		logger.WithError(err).Warnf("Failed to publish factory work order updated for order %s", closed.ID)
	}

	if !wasClosed {
		notification := messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: orgID.String(),
			FactoryID:      factory.ID.String(),
			OrderID:        closed.ID.String(),
			EventType:      factoryevents.EventTypeOrderStatusUpdated,
			ActorUserID:    closedBy.String(),
			FromState:      fromState,
			ToState:        models.FactoryWorkOrderStateClosed,
			Result:         result,
		}
		if err := notification.Publish(); err != nil {
			logger.WithError(err).Warnf("Failed to publish work order notification for order %s", closed.ID)
		}
	}

	return factory.FindWorkOrder(db, closed.ID)
}
