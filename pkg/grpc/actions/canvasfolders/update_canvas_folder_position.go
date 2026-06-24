package canvasfolders

import (
	"context"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
)

func UpdateCanvasFolderPosition(
	_ context.Context,
	organizationID,
	id string,
	direction pb.UpdateCanvasFolderPositionRequest_Direction,
) (*pb.UpdateCanvasFolderPositionResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	folderID, err := uuid.Parse(id)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas folder id")
	}

	if direction == pb.UpdateCanvasFolderPositionRequest_DIRECTION_UNSPECIFIED {
		return nil, grpcerrors.InvalidArgument(nil, "canvas folder move direction is required")
	}

	folders, err := models.MoveCanvasFolder(organizationUUID, folderID, direction.String())
	if err != nil {
		return nil, canvasFolderErrorToStatus(err, "failed to move canvas folder")
	}

	return &pb.UpdateCanvasFolderPositionResponse{
		Folders: SerializeCanvasFolders(folders),
	}, nil
}
