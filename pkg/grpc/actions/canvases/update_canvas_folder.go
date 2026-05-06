package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateCanvasFolder(
	_ context.Context,
	organizationID,
	id string,
	folder *pb.CanvasFolder,
	direction pb.UpdateCanvasFolderRequest_Direction,
) (*pb.UpdateCanvasFolderResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	folderID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas folder id: %v", err)
	}

	if direction != pb.UpdateCanvasFolderRequest_DIRECTION_UNSPECIFIED {
		if folder != nil {
			return nil, status.Error(codes.InvalidArgument, "canvas folder fields cannot be updated while moving a folder")
		}

		folders, err := models.MoveCanvasFolder(organizationUUID, folderID, direction.String())
		if err != nil {
			return nil, canvasFolderErrorToStatus(err)
		}

		return &pb.UpdateCanvasFolderResponse{
			Folders: SerializeCanvasFolders(folders),
		}, nil
	}

	if folder == nil || folder.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas folder is required")
	}

	updatedFolder, err := models.UpdateCanvasFolder(organizationUUID, folderID, folder.Spec.Title, folder.Spec.BackgroundColor)
	if err != nil {
		return nil, canvasFolderErrorToStatus(err)
	}

	return &pb.UpdateCanvasFolderResponse{
		Folder: SerializeCanvasFolder(updatedFolder),
	}, nil
}
