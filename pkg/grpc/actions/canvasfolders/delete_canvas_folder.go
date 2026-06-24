package canvasfolders

import (
	"context"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
)

func DeleteCanvasFolder(_ context.Context, organizationID, id string) (*pb.DeleteCanvasFolderResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	folderID, err := uuid.Parse(id)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas folder id")
	}

	if err := models.DeleteCanvasFolder(organizationUUID, folderID); err != nil {
		return nil, canvasFolderErrorToStatus(err, "failed to delete canvas folder")
	}

	return &pb.DeleteCanvasFolderResponse{}, nil
}
