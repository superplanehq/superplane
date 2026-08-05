package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
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

	kind := commentAuthorKindFromProto(req.GetAuthorKind())
	author := factory.WorkOrderCommentAuthor{
		Kind:  kind,
		Label: strings.TrimSpace(req.GetAuthorLabel()),
	}
	//
	// Only user-authored comments carry the authenticated caller's id. LLM /
	// system comments intentionally omit `UserID` so the timeline doesn't
	// misattribute automated notes to the human that made the API call
	// (matching the canvas-side `addWorkOrderComment` component's shape).
	//
	if kind == factory.CommentAuthorKindUser {
		author.UserID = &userIDStr
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

	return &pb.AddWorkOrderCommentResponse{
		Comment: &pb.WorkOrderComment{
			Body:      body,
			Author:    serializeCommentAuthor(author),
			CreatedAt: timestamppb.Now(),
		},
	}, nil
}

func commentAuthorKindFromProto(kind pb.WorkOrderCommentAuthor_Kind) string {
	switch kind {
	case pb.WorkOrderCommentAuthor_KIND_LLM:
		return factory.CommentAuthorKindLLM
	case pb.WorkOrderCommentAuthor_KIND_SYSTEM:
		return factory.CommentAuthorKindSystem
	default:
		return factory.CommentAuthorKindUser
	}
}

func serializeCommentAuthor(author factory.WorkOrderCommentAuthor) *pb.WorkOrderCommentAuthor {
	result := &pb.WorkOrderCommentAuthor{
		Kind:  commentAuthorKindToProto(author.Kind),
		Label: author.Label,
	}
	if author.UserID != nil {
		result.UserId = *author.UserID
	}
	return result
}

func commentAuthorKindToProto(kind string) pb.WorkOrderCommentAuthor_Kind {
	switch kind {
	case factory.CommentAuthorKindLLM:
		return pb.WorkOrderCommentAuthor_KIND_LLM
	case factory.CommentAuthorKindSystem:
		return pb.WorkOrderCommentAuthor_KIND_SYSTEM
	case factory.CommentAuthorKindUser:
		return pb.WorkOrderCommentAuthor_KIND_USER
	default:
		return pb.WorkOrderCommentAuthor_KIND_UNSPECIFIED
	}
}
