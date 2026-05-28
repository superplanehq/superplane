package canvases

import (
	"bytes"
	"context"
	"io"

	"github.com/superplanehq/superplane/pkg/git"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func CommitCanvasRepositoryFiles(
	ctx context.Context,
	organizationID string,
	canvasID string,
	req *pb.CommitCanvasRepositoryFilesRequest,
	storage git.Provider,
	options CanvasRepositoryStorageOptions,
) (*pb.CommitCanvasRepositoryFilesResponse, error) {
	repository, err := requireReadyCanvasRepository(organizationID, canvasID, storage)
	if err != nil {
		return nil, err
	}

	author, err := canvasRepositoryCommitAuthor(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	operations := make([]git.FileOperation, 0, len(req.GetOperations()))
	for _, operation := range req.GetOperations() {
		content := operation.GetContent()
		var reader io.Reader
		if !operation.GetDelete() {
			reader = bytes.NewReader(content)
		}

		operations = append(operations, git.FileOperation{
			Path:      operation.GetPath(),
			Content:   reader,
			SizeBytes: int64(len(content)),
			Delete:    operation.GetDelete(),
		})
	}

	result, err := storage.Commit(ctx, canvasRepositoryRef(repository), git.CommitOptions{
		Branch:          "main",
		BaseBranch:      "main",
		ExpectedHeadSHA: req.GetExpectedHeadSha(),
		Message:         req.GetMessage(),
		Author:          author,
		Operations:      operations,
	})

	if err != nil {
		return nil, gitStorageStatusError(err)
	}

	return &pb.CommitCanvasRepositoryFilesResponse{
		CommitSha: result.CommitSHA,
	}, nil
}
