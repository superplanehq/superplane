package factories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
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
	var fromState string
	var changed bool
	err = db.Transaction(func(tx *gorm.DB) error {
		factory, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}

		order, err = factory.FindWorkOrder(tx, orderID)
		if err != nil {
			return err
		}

		fromState = order.State
		changed, err = order.UpdateStatus(tx, models.FactoryWorkOrderStatusUpdate{
			ToState: toState,
			Result:  result,
			Actor:   &actor,
		})
		return err
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order status")
	}

	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order status")
	}

	refreshed, err := factory.FindWorkOrder(db, orderID)
	if errors.Is(err, models.ErrFactoryWorkOrderNotFound) {
		serialized, err := serializeWorkOrder(factory, order, nil, nil)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to update work order status")
		}
		return &pb.UpdateWorkOrderStatusResponse{Order: serialized}, nil
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order status")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}

	if changed {
		notification := messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: orgID.String(),
			FactoryID:      factory.ID.String(),
			OrderID:        order.ID.String(),
			EventType:      factoryevents.EventTypeOrderStatusUpdated,
			ActorUserID:    actor.String(),
			FromState:      fromState,
			ToState:        toState,
			Result:         result,
		}
		if err := notification.Publish(); err != nil {
			log.WithError(err).Warnf("Failed to publish work order notification for order %s", order.ID)
		}
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, refreshed)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order status")
	}

	return &pb.UpdateWorkOrderStatusResponse{Order: serialized}, nil
}
