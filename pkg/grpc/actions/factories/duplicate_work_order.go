package factories

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	workersctx "github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

func DuplicateWorkOrder(ctx context.Context, organizationID string, req *pb.DuplicateWorkOrderRequest) (*pb.DuplicateWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to duplicate work order")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to duplicate work order")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to duplicate work order")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	createdByID, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to duplicate work order")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to duplicate work order")
	}

	source, err := factory.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to duplicate work order")
	}

	assigneeIDs := []uuid.UUID{createdByID}
	var order *models.FactoryWorkOrder
	var bound storedfiles.BindResult
	err = db.Transaction(func(tx *gorm.DB) error {
		created, err := createDuplicatedWorkOrder(tx, factory, source, createdByID, assigneeIDs)
		if err != nil {
			return err
		}
		order = created

		if err := order.CopyRepositoryFrom(tx, source); err != nil {
			return err
		}

		cloned, cloneErr := storedfiles.CloneDescriptionFiles(
			ctx,
			tx,
			blob.Current(),
			orgID,
			factory.ID,
			order.ID,
			createdByID,
			source.Description,
		)
		bound.CopiedKeys = cloned.CopiedKeys
		if cloneErr != nil {
			return cloneErr
		}
		if cloned.Markdown != order.Description {
			return order.UpdateContent(tx, nil, &cloned.Markdown)
		}
		return nil
	})
	if delErr := storedfiles.ApplyBindResult(ctx, db, blob.Current(), orgID, factory.ID, bound, err); delErr != nil {
		log.WithError(delErr).Warn("Failed to delete file objects after duplicate")
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to duplicate work order")
	}

	workersctx.EmitWorkOrderCreated(db, factory, order)

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}

	if assignedIDs := newAssigneeIDs(nil, assigneeIDs); len(assignedIDs) > 0 {
		notification := messages.FactoryWorkOrderNotificationMessage{
			OrganizationID:  orgID.String(),
			FactoryID:       factory.ID.String(),
			OrderID:         order.ID.String(),
			EventType:       factoryevents.EventTypeOrderAssigneesUpdated,
			ActorUserID:     createdByID.String(),
			AssignedUserIDs: assignedIDs,
		}
		if err := notification.Publish(); err != nil {
			log.WithError(err).Warnf("Failed to publish work order notification for order %s", order.ID)
		}
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to duplicate work order")
	}

	return &pb.DuplicateWorkOrderResponse{
		Order: serialized,
	}, nil
}

func createDuplicatedWorkOrder(
	tx *gorm.DB,
	factory *models.Factory,
	source *models.FactoryWorkOrder,
	createdByID uuid.UUID,
	assigneeIDs []uuid.UUID,
) (*models.FactoryWorkOrder, error) {
	if origin := source.Origin(); origin != nil {
		return factory.CreateWorkOrderWithOrigin(
			tx,
			source.Title,
			source.Description,
			&createdByID,
			assigneeIDs,
			nil,
			*origin,
		)
	}
	return factory.CreateWorkOrder(tx, source.Title, source.Description, &createdByID, assigneeIDs, nil)
}
