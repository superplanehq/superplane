package factories

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func RemoveWorkOrderCommentReaction(
	ctx context.Context,
	organizationID string,
	req *pb.RemoveWorkOrderCommentReactionRequest,
) (*pb.RemoveWorkOrderCommentReactionResponse, error) {
	orgID, commentID, factoryID, orderID, userID, err := parseCommentReactionRequest(
		ctx, organizationID, req.GetFactoryId(), req.GetOrderId(), req.GetCommentId(), req.GetEmoji(),
	)
	if err != nil {
		return nil, err
	}

	var summaries []models.CommentReactionSummary
	db := database.DB(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		factoryModel, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}

		order, err := factoryModel.FindWorkOrder(tx, orderID)
		if err != nil {
			return err
		}

		if err := order.RemoveCommentReaction(tx, commentID, userID, req.GetEmoji()); err != nil {
			return err
		}

		byComment, err := models.ListCommentReactionSummaries(tx, orderID, []uuid.UUID{commentID}, userID)
		if err != nil {
			return err
		}

		summaries = byComment[commentID]
		return nil
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to remove work order comment reaction")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		orderID.String(),
		factory.EventTypeOrderCommentReactionUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", orderID)
	}

	return &pb.RemoveWorkOrderCommentReactionResponse{
		Reactions: serializeCommentReactionSummaries(summaries),
	}, nil
}
