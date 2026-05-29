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

	repository, err := models.FindRepository(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find repository for canvas %s: %v", canvasID, err)
	}

	files, err := gitProvider.ListFiles(ctx, repository.RepoID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get repository head sha: %v", err)
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
