package canvasfolders

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
)

func ListCanvasFolders(_ context.Context, organizationID string) (*pb.ListCanvasFoldersResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	folders, err := models.ListCanvasFolders(organizationUUID)
	if err != nil {
		log.Errorf("failed to list canvas folders for organization %s: %v", organizationID, err)
		return nil, grpcerrors.Internal(err, "failed to list canvas folders")
	}

	return &pb.ListCanvasFoldersResponse{
		Folders: SerializeCanvasFolders(folders),
	}, nil
}
