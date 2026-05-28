package canvases

import (
	"context"
	"io"
	"net/url"

	"github.com/superplanehq/superplane/pkg/git"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OpenedCanvasRepositoryFile struct {
	Path    string
	Content io.ReadCloser
}

func OpenCanvasRepositoryFile(
	ctx context.Context,
	organizationID string,
	canvasID string,
	path string,
	ref string,
	storage git.Provider,
	options CanvasRepositoryStorageOptions,
) (*OpenedCanvasRepositoryFile, error) {
	repository, err := requireReadyCanvasRepository(organizationID, canvasID, storage)
	if err != nil {
		return nil, err
	}

	requestPath, err := url.PathUnescape(path)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file path")
	}

	normalizedPath, err := git.NormalizePath(requestPath)
	if err != nil {
		return nil, gitStorageStatusError(err)
	}

	reader, err := storage.GetFile(ctx, canvasRepositoryRef(repository), git.GetFileOptions{
		Path: normalizedPath,
		Ref:  ref,
	})
	if err != nil {
		return nil, gitStorageStatusError(err)
	}

	return &OpenedCanvasRepositoryFile{
		Path:    normalizedPath,
		Content: reader,
	}, nil
}
