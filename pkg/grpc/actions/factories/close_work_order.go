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
	"github.com/superplanehq/superplane/pkg/productanalytics"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func CloseWorkOrder(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.CloseWorkOrderRequest,
) (*pb.CloseWorkOrderResponse, error) {
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

	if result == models.FactoryWorkOrderResultRejected {
		result = models.FactoryWorkOrderResultManualRejected
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	logger := logging.ForFactory(*factory)
	order, err := findWorkOrder(db, factory, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}
	orderID := order.ID

	logger = logging.WithWorkOrder(logger, *order)
	fromState := order.State
	wasClosed := order.IsClosed()
	order, err = order.Close(db, result, &closedBy)
	if err != nil {
		logger.WithError(err).Error("close work order failed")
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	publishWorkOrderClosed(orgID, factory, order, &closedBy, fromState, result, wasClosed)
	if !wasClosed {
		if eventName, properties, ok := factoryCloseEvent(result); ok {
			captureFactoryProductEvent(
				ctx,
				db,
				factory,
				order.ID,
				eventName,
				productanalytics.StableEventID(eventName, order.ID.String(), result),
				properties,
			)
		}
	}
	messages.PublishWorkOrderRunCancellations(db, order)

	if err := closePullRequestsForRejectedWorkOrder(ctx, deps, factory, order); err != nil {
		logger.WithError(err).Error("close work order pull requests failed")
		return nil, factoryErrorToStatus(err, "failed to close work order")
	}

	order, err = factory.FindWorkOrder(db, orderID)
	if err != nil {
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

func publishWorkOrderClosed(
	orgID uuid.UUID,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	closedBy *uuid.UUID,
	fromState string,
	result string,
	wasClosed bool,
) {
	logger := logging.WithWorkOrder(logging.ForFactory(*factory), *order)
	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		logger.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}

	if wasClosed {
		return
	}

	notification := messages.FactoryWorkOrderNotificationMessage{
		OrganizationID: orgID.String(),
		FactoryID:      factory.ID.String(),
		OrderID:        order.ID.String(),
		EventType:      factoryevents.EventTypeOrderStatusUpdated,
		FromState:      fromState,
		ToState:        models.FactoryWorkOrderStateClosed,
		Result:         result,
	}
	if closedBy != nil {
		notification.ActorUserID = closedBy.String()
	}
	if err := notification.Publish(); err != nil {
		logger.WithError(err).Warnf("Failed to publish work order notification for order %s", order.ID)
	}
}

func closePullRequestsForRejectedWorkOrder(
	ctx context.Context,
	deps IntakeDependencies,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
) error {
	if order == nil {
		return nil
	}
	return withFactoryWorkOrderLock(ctx, order.ID, func(locked *gorm.DB) error {
		current, err := factory.FindWorkOrder(locked, order.ID)
		if err != nil {
			return err
		}
		if !workOrderResultClosesPullRequests(current.Result) {
			return nil
		}

		grouped, err := models.ListPullRequestsByWorkOrderIDs(locked, []uuid.UUID{order.ID})
		if err != nil {
			return err
		}
		closeable := closeablePullRequests(grouped[order.ID])
		if len(closeable) == 0 {
			return nil
		}
		return closePreviousPullRequests(ctx, locked, deps, factory, closeable)
	})
}

func withFactoryWorkOrderLock(ctx context.Context, id uuid.UUID, fn func(db *gorm.DB) error) error {
	sqlDB, err := database.Conn().DB()
	if err != nil {
		return err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	key := models.FactoryWorkOrderLockKey(id)
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock(hashtextextended($1, 0))", key); err != nil {
		return err
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock(hashtextextended($1, 0))", key)
	}()

	locked, err := gorm.Open(postgres.New(postgres.Config{Conn: conn}), &gorm.Config{})
	if err != nil {
		return err
	}
	return fn(locked.WithContext(ctx))
}

func workOrderResultClosesPullRequests(result string) bool {
	switch result {
	case models.FactoryWorkOrderResultRejected,
		models.FactoryWorkOrderResultManualRejected,
		models.FactoryWorkOrderResultArchived:
		return true
	default:
		return false
	}
}
