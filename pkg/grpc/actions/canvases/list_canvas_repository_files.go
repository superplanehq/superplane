package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
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

	_, err = models.FindCanvas(orgID, canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to load canvas: %v", err)
	}

	repositoryPaths := []string{}
	repository, err := models.FindRepository(orgID, canvasID)
	switch {
	case err == nil && repository.Status == models.RepositoryStatusReady:
		//
		// Only call out to the git provider when the repository has been
		// successfully provisioned. For pending/error states the underlying
		// repo does not yet exist on the git provider side, so calling
		// ListFiles() would fail and produce a 500.
		//
		files, listErr := gitProvider.ListFiles(ctx, repository.RepoID)
		if listErr != nil {
			return nil, status.Errorf(codes.Internal, "failed to list repository files: %v", listErr)
		}
		repositoryPaths = files
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, status.Errorf(codes.Internal, "failed to load repository: %v", err)
	}

	pathStrings := AppendRepositorySpecFilePaths(repositoryPaths)
	paths := make([]*pb.CanvasRepositoryFile, 0, len(pathStrings))
	for _, path := range pathStrings {
		paths = append(paths, &pb.CanvasRepositoryFile{
			Path: path,
		})
	}

	return &pb.ListCanvasRepositoryFilesResponse{
		Files: paths,
	}, nil
}
