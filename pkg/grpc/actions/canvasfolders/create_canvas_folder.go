package canvasfolders

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
