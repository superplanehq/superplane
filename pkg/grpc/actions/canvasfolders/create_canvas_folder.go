package canvasfolders

import (
	"context"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
)

func CreateCanvasFolder(_ context.Context, organizationID string, folder *pb.CanvasFolder) (*pb.CreateCanvasFolderResponse, error) {
	if folder == nil || folder.Spec == nil {
		return nil, grpcerrors.InvalidArgument(nil, "canvas folder is required")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	createdFolder, err := models.CreateCanvasFolder(organizationUUID, folder.Spec.Title, folder.Spec.BackgroundColor)
	if err != nil {
		return nil, canvasFolderErrorToStatus(err, "failed to create canvas folder")
	}

	return &pb.CreateCanvasFolderResponse{
		Folder: SerializeCanvasFolder(createdFolder),
	}, nil
}
