package canvasfolders

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
