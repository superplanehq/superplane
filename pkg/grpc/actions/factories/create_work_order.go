package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	workersctx "github.com/superplanehq/superplane/pkg/workers/contexts"
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

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	createdByID, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to create work order")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	assigneeIDs := []uuid.UUID{createdByID}
	order, err := factory.CreateWorkOrder(db, title, req.GetDescription(), &createdByID, assigneeIDs, nil)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
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
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	return &pb.CreateWorkOrderResponse{
		Order: serialized,
	}, nil
}
