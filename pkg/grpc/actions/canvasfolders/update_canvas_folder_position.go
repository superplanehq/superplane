package canvasfolders

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
		return nil, canvasFolderErrorToStatus(err, "failed to move canvas folder")
	}

	return &pb.UpdateCanvasFolderPositionResponse{
		Folders: SerializeCanvasFolders(folders),
	}, nil
}
