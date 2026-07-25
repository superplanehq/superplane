package canvases

import (
	"context"
	"errors"
	"sort"

	git "github.com/superplanehq/superplane/pkg/git/provider"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func ListCanvasRepositoryFiles(ctx context.Context, gitProvider git.Provider, canvas *models.Canvas) (*pb.ListCanvasRepositoryFilesResponse, error) {
	repositoryPaths := []string{}
	repository, err := models.FindRepository(canvas.OrganizationID, canvas.ID)
	if err == nil {
		files, listErr := gitProvider.ListFiles(ctx, repository.RepoID, "")
		if listErr != nil {
			return nil, grpcerrors.Internal(listErr, "failed to list repository files")
		}
		repositoryPaths = files
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, grpcerrors.Internal(err, "failed to load repository")
	}

	pathStrings := appendRepositorySpecFilePaths(repositoryPaths)
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

func appendRepositorySpecFilePaths(paths []string) []string {
	merged := make([]string, 0, len(paths)+2)
	seen := make(map[string]struct{}, len(paths)+2)

	for _, specPath := range []string{CanvasYAMLRepositoryPath, ConsoleYAMLRepositoryPath} {
		merged = append(merged, specPath)
		seen[specPath] = struct{}{}
	}

	for _, path := range paths {
		normalized := normalizeRepositoryFilePath(path)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		merged = append(merged, normalized)
	}

	sort.Strings(merged)
	return merged
}
