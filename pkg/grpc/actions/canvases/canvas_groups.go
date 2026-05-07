package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func ListCanvasGroups(_ context.Context, organizationID string) (*pb.ListCanvasGroupsResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	groups, err := models.ListCanvasGroups(organizationUUID)
	if err != nil {
		log.Errorf("failed to list canvas groups for organization %s: %v", organizationID, err)
		return nil, status.Error(codes.Internal, "failed to list canvas groups")
	}

	return &pb.ListCanvasGroupsResponse{
		Groups: SerializeCanvasGroups(groups),
	}, nil
}

func CreateCanvasGroup(_ context.Context, organizationID string, group *pb.CanvasGroup) (*pb.CreateCanvasGroupResponse, error) {
	if group == nil || group.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas group is required")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	createdGroup, err := models.CreateCanvasGroup(organizationUUID, group.Spec.Title, group.Spec.BackgroundColor)
	if err != nil {
		return nil, canvasGroupErrorToStatus(err)
	}

	return &pb.CreateCanvasGroupResponse{
		Group: SerializeCanvasGroup(createdGroup),
	}, nil
}

func UpdateCanvasGroup(_ context.Context, organizationID, id string, group *pb.CanvasGroup) (*pb.UpdateCanvasGroupResponse, error) {
	if group == nil || group.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas group is required")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	groupID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas group id: %v", err)
	}

	updatedGroup, err := models.UpdateCanvasGroup(organizationUUID, groupID, group.Spec.Title, group.Spec.BackgroundColor)
	if err != nil {
		return nil, canvasGroupErrorToStatus(err)
	}

	return &pb.UpdateCanvasGroupResponse{
		Group: SerializeCanvasGroup(updatedGroup),
	}, nil
}

func DeleteCanvasGroup(_ context.Context, organizationID, id string) (*pb.DeleteCanvasGroupResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	groupID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas group id: %v", err)
	}

	if err := models.DeleteCanvasGroup(organizationUUID, groupID); err != nil {
		return nil, canvasGroupErrorToStatus(err)
	}

	return &pb.DeleteCanvasGroupResponse{}, nil
}

func UpdateCanvasGroupPosition(
	_ context.Context,
	organizationID,
	id string,
	direction pb.UpdateCanvasGroupPositionRequest_Direction,
) (*pb.UpdateCanvasGroupPositionResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	groupID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas group id: %v", err)
	}

	if direction == pb.UpdateCanvasGroupPositionRequest_DIRECTION_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "canvas group move direction is required")
	}

	groups, err := models.MoveCanvasGroup(organizationUUID, groupID, direction.String())
	if err != nil {
		return nil, canvasGroupErrorToStatus(err)
	}

	return &pb.UpdateCanvasGroupPositionResponse{
		Groups: SerializeCanvasGroups(groups),
	}, nil
}

func UpdateCanvasGroupMembership(
	_ context.Context,
	organizationID,
	canvasID,
	groupID string,
) (*pb.UpdateCanvasGroupMembershipResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	var groupUUID *uuid.UUID
	if groupID != "" {
		parsedGroupID, err := uuid.Parse(groupID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid canvas group id: %v", err)
		}
		groupUUID = &parsedGroupID
	}

	canvas, err := models.UpdateCanvasGroupMembership(organizationUUID, canvasUUID, groupUUID)
	if err != nil {
		return nil, canvasGroupErrorToStatus(err)
	}

	if publishErr := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); publishErr != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", publishErr)
	}

	var user *models.User
	if canvas.CreatedBy != nil {
		user, err = models.FindMaybeDeletedUserByID(canvas.OrganizationID.String(), canvas.CreatedBy.String())
		if err != nil {
			return nil, err
		}
	}

	serializedCanvas, err := SerializeCanvas(canvas, false, user)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize canvas")
	}

	return &pb.UpdateCanvasGroupMembershipResponse{
		Canvas: serializedCanvas,
	}, nil
}

func canvasGroupErrorToStatus(err error) error {
	switch {
	case errors.Is(err, models.ErrCanvasGroupTitleRequired):
		return status.Error(codes.InvalidArgument, "canvas group title is required")
	case errors.Is(err, models.ErrCanvasGroupTitleTooLong):
		return status.Error(codes.InvalidArgument, "canvas group title must be 128 characters or less")
	case errors.Is(err, models.ErrCanvasGroupInvalidBackgroundColor):
		return status.Error(codes.InvalidArgument, "invalid canvas group background color")
	case errors.Is(err, models.ErrCanvasGroupInvalidMoveDirection):
		return status.Error(codes.InvalidArgument, "invalid canvas group move direction")
	case errors.Is(err, models.ErrCanvasGroupTitleAlreadyExists):
		return status.Error(codes.AlreadyExists, "canvas group with the same title already exists")
	case errors.Is(err, gorm.ErrRecordNotFound):
		return status.Error(codes.NotFound, "canvas group not found")
	default:
		return status.Error(codes.Internal, "failed to update canvas group")
	}
}
