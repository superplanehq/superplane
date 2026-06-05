package canvases

import (
	"context"

	"github.com/google/uuid"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListCanvasRepositoryFiles(ctx context.Context, gitProvider git.Provider, organizationID string, id string) (*pb.ListCanvasRepositoryFilesResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	repository, err := models.FindRepository(orgID, canvasID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "repository not found: %v", err)
	}

	files, err := gitProvider.ListFiles(ctx, repository.RepoID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list repository files: %v", err)
	}

	paths := make([]*pb.CanvasRepositoryFile, 0, len(files))
	for _, path := range files {
		paths = append(paths, &pb.CanvasRepositoryFile{
			Path: path,
		})
	}

	return &pb.ListCanvasRepositoryFilesResponse{
		Files: paths,
	}, nil
}
