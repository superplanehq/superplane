package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func ListCanvasRepositoryFiles(ctx context.Context, gitProvider git.Provider, organizationID string, id string) (*pb.ListCanvasRepositoryFilesResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	_, err = models.FindCanvas(orgID, canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "canvas not found")
		}
		return nil, grpcerrors.Internal(err, "failed to load canvas")
	}

	repositoryPaths := []string{}
	repository, err := models.FindRepository(orgID, canvasID)
	if err == nil {
		files, listErr := gitProvider.ListFiles(ctx, repository.RepoID, "")
		if listErr != nil {
			return nil, grpcerrors.Internal(listErr, "failed to list repository files")
		}
		repositoryPaths = files
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, grpcerrors.Internal(err, "failed to load repository")
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
