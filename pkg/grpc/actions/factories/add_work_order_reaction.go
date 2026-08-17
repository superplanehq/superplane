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

// AddWorkOrderReaction adds the caller's emoji reaction to a work order.
// Reactions are current-state, not history (like assignees, and like the
// PR artifact state in FactoryContext.UpdateWorkOrderArtifact): adding the
// same (user, content) pair twice is a no-op, and no timeline event is
// recorded — only a websocket-only notification when something actually
// changed.
func AddWorkOrderReaction(
	ctx context.Context,
	organizationID string,
	req *pb.AddWorkOrderReactionRequest,
) (*pb.AddWorkOrderReactionResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order reaction")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order reaction")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order reaction")
	}

	content := req.GetContent()
	if !models.IsValidWorkOrderReactionContent(content) {
		return nil, factoryErrorToStatus(invalidArgument("invalid reaction content"), "failed to add work order reaction")
	}

	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to add work order reaction")
	}

	db := database.DB(ctx)
	var added bool
	err = db.Transaction(func(tx *gorm.DB) error {
		factoryModel, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}

		order, err := factoryModel.FindWorkOrder(tx, orderID)
		if err != nil {
			return err
		}

		added, err = order.AddReaction(tx, userID, content)
		return err
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order reaction")
	}

	if added {
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
		return nil, factoryErrorToStatus(err, "failed to add work order reaction")
	}

	return &pb.AddWorkOrderReactionResponse{Reactions: reactions}, nil
}

// loadWorkOrderReactions re-loads the order's reaction rollup after a
// mutation, from the caller's point of view (`reacted_by_me`).
func loadWorkOrderReactions(
	db *gorm.DB,
	orgID, factoryID, orderID, currentUserID uuid.UUID,
) ([]*pb.WorkOrderReaction, error) {
	factoryModel, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, err
	}

	order, err := factoryModel.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, err
	}

	reactions, err := order.ListReactions(db)
	if err != nil {
		return nil, err
	}

	return serializeWorkOrderReactions(reactions, currentUserID), nil
}
