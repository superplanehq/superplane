package canvases

import (
	"bytes"
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CommitCanvasRepositoryFiles(
	ctx context.Context,
	gitProvider git.Provider,
	organizationID string,
	id string,
	expectedHeadSha string,
	message string,
	operations []*pb.CanvasRepositoryFileOperation,
) (*pb.CommitCanvasRepositoryFilesResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

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

	user, err := models.FindActiveUserByID(organizationID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user: %v", err)
	}

	gitOperations := make([]git.FileOperation, 0, len(operations))
	for _, operation := range operations {
		content := operation.GetContent()
		var reader io.Reader
		if !operation.GetDelete() {
			reader = bytes.NewReader(content)
		}

		gitOperations = append(gitOperations, git.FileOperation{
			Path:      operation.GetPath(),
			Content:   reader,
			SizeBytes: int64(len(content)),
			Delete:    operation.GetDelete(),
		})
	}

	newCommitSha, err := gitProvider.Commit(ctx, repository.RepoID, git.CommitOptions{
		Branch:          "main",
		BaseBranch:      "main",
		ExpectedHeadSHA: expectedHeadSha,
		Message:         message,
		Operations:      gitOperations,
		Author: git.CommitAuthor{
			Name:  user.Name,
			Email: user.GetEmail(),
		},
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit repository files: %v", err)
	}

	return &pb.CommitCanvasRepositoryFilesResponse{
		CommitSha: newCommitSha,
	}, nil
}
