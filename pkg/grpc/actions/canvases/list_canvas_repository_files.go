package canvases

import (
	"context"

	"github.com/superplanehq/superplane/pkg/git"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func ListCanvasRepositoryFiles(
	ctx context.Context,
	organizationID string,
	canvasID string,
	storage git.Provider,
	options CanvasRepositoryStorageOptions,
) (*pb.ListCanvasRepositoryFilesResponse, error) {
	repository, err := requireReadyCanvasRepository(organizationID, canvasID, storage)
	if err != nil {
		return nil, err
	}

	result, err := storage.ListFiles(ctx, canvasRepositoryRef(repository), git.ListFilesOptions{Ref: "main"})
	if err != nil {
		return nil, gitStorageStatusError(err)
	}

	files := make([]*pb.CanvasRepositoryFile, 0, len(result.Paths))
	for _, path := range result.Paths {
		files = append(files, &pb.CanvasRepositoryFile{Path: path})
	}

	return &pb.ListCanvasRepositoryFilesResponse{
		Files: files,
	}, nil
}
