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
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func AddWorkOrderComment(
	ctx context.Context,
	organizationID string,
	req *pb.AddWorkOrderCommentRequest,
) (*pb.AddWorkOrderCommentResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order comment")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order comment")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order comment")
	}

	body := strings.TrimSpace(req.GetBody())
	if body == "" {
		return nil, factoryErrorToStatus(invalidArgument("body is required"), "failed to add work order comment")
	}

	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	if _, err := uuid.Parse(userIDStr); err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to add work order comment")
	}

	// Interactive endpoint: author is always the caller. Automation
	// comments arrive through the canvas component with its own kind.
	author := factory.WorkOrderCommentAuthor{
		Kind:   factory.CommentAuthorKindUser,
		UserID: &userIDStr,
	}

	var comment *models.FactoryWorkOrderComment
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

		comment, err = order.RecordCommentAdded(tx, models.FactoryWorkOrderCommentParams{
			Body:             body,
			Author:           author,
			MentionedUserIDs: parseMentionedUserIDs(req.GetMentionedUserIds()),
		})
		return err
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order comment")
	}

	publishWorkOrderCommentNotifications(factoryID, orderID, orgID, userIDStr, comment)

	return &pb.AddWorkOrderCommentResponse{
		Comment: serializeWorkOrderComment(comment),
	}, nil
}

func publishWorkOrderCommentNotifications(
	factoryID, orderID, orgID uuid.UUID,
	actorUserID string,
	comment *models.FactoryWorkOrderComment,
) {
	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		orderID.String(),
		factory.EventTypeOrderCommentAdded,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", orderID)
	}

	notification := messages.FactoryWorkOrderNotificationMessage{
		OrganizationID:   orgID.String(),
		FactoryID:        factoryID.String(),
		OrderID:          orderID.String(),
		EventType:        factory.EventTypeOrderCommentAdded,
		ActorUserID:      actorUserID,
		CommentBody:      comment.Body,
		MentionedUserIDs: uuidStrings(comment.MentionedUserIDs),
	}
	if err := notification.Publish(); err != nil {
		log.WithError(err).Warnf("Failed to publish work order notification for order %s", orderID)
	}
}

func serializeWorkOrderComment(comment *models.FactoryWorkOrderComment) *pb.WorkOrderComment {
	return &pb.WorkOrderComment{
		Id:               comment.ID.String(),
		Body:             comment.Body,
		Author:           serializeCommentAuthor(comment.Author()),
		CreatedAt:        timestamppb.New(comment.CreatedAt),
		MentionedUserIds: uuidStrings(comment.MentionedUserIDs),
	}
}

func serializeCommentAuthor(author factory.WorkOrderCommentAuthor) *pb.WorkOrderCommentAuthor {
	result := &pb.WorkOrderCommentAuthor{
		Kind: commentAuthorKindToProto(author.Kind),
	}
	if author.UserID != nil {
		result.UserId = *author.UserID
	}
	if author.Automation != nil {
		result.Automation = &pb.AutomationRef{
			NodeId:   author.Automation.NodeID,
			NodeName: author.Automation.NodeName,
			AppId:    author.Automation.AppID.String(),
			AppName:  author.Automation.AppName,
		}
	}
	return result
}

func commentAuthorKindToProto(kind string) pb.WorkOrderCommentAuthor_Kind {
	switch kind {
	case factory.CommentAuthorKindAutomation:
		return pb.WorkOrderCommentAuthor_KIND_AUTOMATION
	case factory.CommentAuthorKindUser:
		return pb.WorkOrderCommentAuthor_KIND_USER
	default:
		return pb.WorkOrderCommentAuthor_KIND_UNSPECIFIED
	}
}

func parseMentionedUserIDs(rawIDs []string) []uuid.UUID {
	mentioned := make([]uuid.UUID, 0, len(rawIDs))
	for _, rawID := range rawIDs {
		userID, err := uuid.Parse(rawID)
		if err != nil {
			continue
		}
		mentioned = append(mentioned, userID)
	}
	return mentioned
}

func uuidStrings(ids []uuid.UUID) []string {
	if len(ids) == 0 {
		return nil
	}

	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, id.String())
	}
	return result
}
