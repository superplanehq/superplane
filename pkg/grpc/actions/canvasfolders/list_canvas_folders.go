package canvasfolders

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
