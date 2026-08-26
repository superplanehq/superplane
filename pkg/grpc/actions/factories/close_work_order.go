package factories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
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
	fromState := order.State
	wasClosed := order.IsClosed()

	order, err = order.Close(db, result, &closedBy)
	if err != nil {
		logger.WithError(err).Error("close work order failed")
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	refreshed, err := factory.FindWorkOrder(db, orderID)
	if errors.Is(err, models.ErrFactoryWorkOrderNotFound) {
		serialized, err := serializeWorkOrder(factory, order, nil, nil)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to close work order")
		}
		return &pb.CloseWorkOrderResponse{Order: serialized}, nil
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		logger.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}

	if !wasClosed {
		notification := messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: orgID.String(),
			FactoryID:      factory.ID.String(),
			OrderID:        order.ID.String(),
			EventType:      factoryevents.EventTypeOrderStatusUpdated,
			ActorUserID:    closedBy.String(),
			FromState:      fromState,
			ToState:        models.FactoryWorkOrderStateClosed,
			Result:         result,
		}
		if err := notification.Publish(); err != nil {
			logger.WithError(err).Warnf("Failed to publish work order notification for order %s", order.ID)
		}
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, refreshed)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	return &pb.CloseWorkOrderResponse{
		Order: serialized,
	}, nil
}
