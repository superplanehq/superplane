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

func AddWorkOrderCommentReaction(
	ctx context.Context,
	organizationID string,
	req *pb.AddWorkOrderCommentReactionRequest,
) (*pb.AddWorkOrderCommentReactionResponse, error) {
	orgID, commentID, factoryID, orderID, userID, err := parseCommentReactionRequest(
		ctx, organizationID, req.GetFactoryId(), req.GetOrderId(), req.GetCommentId(), req.GetEmoji(),
	)
	if err != nil {
		// parseCommentReactionRequest already returns fully-formed gRPC
		// status errors (invalid argument / unauthenticated), unlike the
		// model/transaction errors below that still need mapping.
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

		if err := order.AddCommentReaction(tx, commentID, userID, req.GetEmoji()); err != nil {
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
		return nil, factoryErrorToStatus(err, "failed to add work order comment reaction")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		orderID.String(),
		factory.EventTypeOrderCommentReactionUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", orderID)
	}

	return &pb.AddWorkOrderCommentReactionResponse{
		Reactions: serializeCommentReactionSummaries(summaries),
	}, nil
}

// parseCommentReactionRequest holds the parsing/validation shared by
// AddWorkOrderCommentReaction and RemoveWorkOrderCommentReaction. Unlike
// the model/transaction errors those callers map through
// factoryErrorToStatus, every error returned here is already a
// fully-formed gRPC status error.
func parseCommentReactionRequest(
	ctx context.Context,
	organizationID, factoryIDStr, orderIDStr, commentIDStr, emoji string,
) (orgID, commentID, factoryID, orderID, userID uuid.UUID, err error) {
	orgID, err = parseOrganizationID(organizationID)
	if err != nil {
		err = grpcerrors.InvalidArgument(err, err.Error())
		return
	}

	factoryID, err = parseFactoryID(factoryIDStr)
	if err != nil {
		err = grpcerrors.InvalidArgument(err, err.Error())
		return
	}

	orderID, err = parseOrderID(orderIDStr)
	if err != nil {
		err = grpcerrors.InvalidArgument(err, err.Error())
		return
	}

	commentID, err = uuid.Parse(commentIDStr)
	if err != nil {
		err = grpcerrors.InvalidArgument(err, "invalid comment id")
		return
	}

	if !models.IsValidCommentReactionEmoji(emoji) {
		err = grpcerrors.InvalidArgument(invalidArgument("invalid reaction emoji"), "invalid reaction emoji")
		return
	}

	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		err = grpcerrors.Unauthenticated(nil, "user not authenticated")
		return
	}
	userID, parseErr := uuid.Parse(userIDStr)
	if parseErr != nil {
		err = grpcerrors.InvalidArgument(parseErr, "invalid user id")
		return
	}

	return
}

func serializeCommentReactionSummaries(summaries []models.CommentReactionSummary) []*pb.WorkOrderCommentReactionSummary {
	result := make([]*pb.WorkOrderCommentReactionSummary, 0, len(summaries))
	for _, s := range summaries {
		result = append(result, &pb.WorkOrderCommentReactionSummary{
			Emoji:       s.Emoji,
			Count:       int32(s.Count),
			ReactedByMe: s.ReactedByMe,
		})
	}
	return result
}
