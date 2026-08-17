package factories

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

// RemoveWorkOrderReaction removes the caller's emoji reaction from a work
// order. Symmetric to AddWorkOrderReaction: removing a reaction that was
// never added (or already removed) is a no-op, and only publishes the
// websocket-only notification when a row was actually deleted.
func RemoveWorkOrderReaction(
	ctx context.Context,
	organizationID string,
	req *pb.RemoveWorkOrderReactionRequest,
) (*pb.RemoveWorkOrderReactionResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove work order reaction")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove work order reaction")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove work order reaction")
	}

	content := req.GetContent()
	if !models.IsValidWorkOrderReactionContent(content) {
		return nil, factoryErrorToStatus(invalidArgument("invalid reaction content"), "failed to remove work order reaction")
	}

	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to remove work order reaction")
	}

	db := database.DB(ctx)
	var removed bool
	err = db.Transaction(func(tx *gorm.DB) error {
		factoryModel, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}

		order, err := factoryModel.FindWorkOrder(tx, orderID)
		if err != nil {
			return err
		}

		removed, err = order.RemoveReaction(tx, userID, content)
		return err
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove work order reaction")
	}

	if removed {
		if err := messages.PublishFactoryWorkOrderUpdated(
			factoryID.String(),
			orderID.String(),
			factory.EventTypeOrderReactionUpdated,
		); err != nil {
			log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", orderID)
		}
	}

	reactions, err := loadWorkOrderReactions(db, orgID, factoryID, orderID, userID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove work order reaction")
	}

	return &pb.RemoveWorkOrderReactionResponse{Reactions: reactions}, nil
}
