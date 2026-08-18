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

		return order.RecordCommentAdded(tx, body, author, nil)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to add work order comment")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		orderID.String(),
		factory.EventTypeOrderCommentAdded,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", orderID)
	}

	notification := messages.FactoryWorkOrderNotificationMessage{
		OrganizationID: orgID.String(),
		FactoryID:      factoryID.String(),
		OrderID:        orderID.String(),
		EventType:      factory.EventTypeOrderCommentAdded,
		ActorUserID:    userIDStr,
		CommentBody:    body,
	}
	if err := notification.Publish(); err != nil {
		log.WithError(err).Warnf("Failed to publish work order notification for order %s", orderID)
	}

	return &pb.AddWorkOrderCommentResponse{
		Comment: &pb.WorkOrderComment{
			Body:      body,
			Author:    serializeCommentAuthor(author),
			CreatedAt: timestamppb.Now(),
		},
	}, nil
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
