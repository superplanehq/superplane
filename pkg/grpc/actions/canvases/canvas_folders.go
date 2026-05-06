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

func ListCanvasFolders(_ context.Context, organizationID string) (*pb.ListCanvasFoldersResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	folders, err := models.ListCanvasFolders(organizationUUID)
	if err != nil {
		log.Errorf("failed to list canvas folders for organization %s: %v", organizationID, err)
		return nil, status.Error(codes.Internal, "failed to list canvas folders")
	}

	return &pb.ListCanvasFoldersResponse{
		Folders: SerializeCanvasFolders(folders),
	}, nil
}

func CreateCanvasFolder(_ context.Context, organizationID string, folder *pb.CanvasFolder) (*pb.CreateCanvasFolderResponse, error) {
	if folder == nil || folder.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas folder is required")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	createdFolder, err := models.CreateCanvasFolder(organizationUUID, folder.Spec.Title, folder.Spec.BackgroundColor)
	if err != nil {
		return nil, canvasFolderErrorToStatus(err)
	}

	return &pb.CreateCanvasFolderResponse{
		Folder: SerializeCanvasFolder(createdFolder),
	}, nil
}

func UpdateCanvasFolder(_ context.Context, organizationID, id string, folder *pb.CanvasFolder) (*pb.UpdateCanvasFolderResponse, error) {
	if folder == nil || folder.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas folder is required")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	folderID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas folder id: %v", err)
	}

	updatedFolder, err := models.UpdateCanvasFolder(organizationUUID, folderID, folder.Spec.Title, folder.Spec.BackgroundColor)
	if err != nil {
		return nil, canvasFolderErrorToStatus(err)
	}

	return &pb.UpdateCanvasFolderResponse{
		Folder: SerializeCanvasFolder(updatedFolder),
	}, nil
}

func DeleteCanvasFolder(_ context.Context, organizationID, id string) (*pb.DeleteCanvasFolderResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	folderID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas folder id: %v", err)
	}

	if err := models.DeleteCanvasFolder(organizationUUID, folderID); err != nil {
		return nil, canvasFolderErrorToStatus(err)
	}

	return &pb.DeleteCanvasFolderResponse{}, nil
}

func UpdateCanvasFolderPosition(
	_ context.Context,
	organizationID,
	id string,
	direction pb.UpdateCanvasFolderPositionRequest_Direction,
) (*pb.UpdateCanvasFolderPositionResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	folderID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas folder id: %v", err)
	}

	if direction == pb.UpdateCanvasFolderPositionRequest_DIRECTION_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "canvas folder move direction is required")
	}

	folders, err := models.MoveCanvasFolder(organizationUUID, folderID, direction.String())
	if err != nil {
		return nil, canvasFolderErrorToStatus(err)
	}

	return &pb.UpdateCanvasFolderPositionResponse{
		Folders: SerializeCanvasFolders(folders),
	}, nil
}

func UpdateCanvasFolderMembership(
	_ context.Context,
	organizationID,
	canvasID,
	folderID string,
) (*pb.UpdateCanvasFolderMembershipResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	var folderUUID *uuid.UUID
	if folderID != "" {
		parsedFolderID, err := uuid.Parse(folderID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid canvas folder id: %v", err)
		}
		folderUUID = &parsedFolderID
	}

	canvas, err := models.UpdateCanvasFolderMembership(organizationUUID, canvasUUID, folderUUID)
	if err != nil {
		return nil, canvasFolderErrorToStatus(err)
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

	return &pb.UpdateCanvasFolderMembershipResponse{
		Canvas: serializedCanvas,
	}, nil
}

func canvasFolderErrorToStatus(err error) error {
	switch {
	case errors.Is(err, models.ErrCanvasFolderTitleRequired):
		return status.Error(codes.InvalidArgument, "canvas folder title is required")
	case errors.Is(err, models.ErrCanvasFolderTitleTooLong):
		return status.Error(codes.InvalidArgument, "canvas folder title must be 128 characters or less")
	case errors.Is(err, models.ErrCanvasFolderInvalidBackgroundColor):
		return status.Error(codes.InvalidArgument, "invalid canvas folder background color")
	case errors.Is(err, models.ErrCanvasFolderInvalidMoveDirection):
		return status.Error(codes.InvalidArgument, "invalid canvas folder move direction")
	case errors.Is(err, models.ErrCanvasFolderTitleAlreadyExists):
		return status.Error(codes.AlreadyExists, "canvas folder with the same title already exists")
	case errors.Is(err, gorm.ErrRecordNotFound):
		return status.Error(codes.NotFound, "canvas folder not found")
	default:
		return status.Error(codes.Internal, "failed to update canvas folder")
	}
}
