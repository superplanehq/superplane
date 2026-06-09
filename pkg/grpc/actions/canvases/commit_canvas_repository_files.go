package canvases

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func CommitCanvasRepositoryFiles(
	ctx context.Context,
	gitProvider git.Provider,
	usageService usage.Service,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	id string,
	versionID string,
	expectedHeadSha string,
	message string,
	operations []*pb.CanvasRepositoryFileOperation,
	autoLayout *pb.CanvasAutoLayout,
	webhookBaseURL string,
	authService authorization.Authorization,
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

	_, err = models.FindCanvas(orgID, canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to load canvas: %v", err)
	}

	resolvedAutoLayout := resolveCommitCanvasAutoLayout(autoLayout != nil, autoLayout)
	specOps, gitOps := splitRepositoryFileOperations(operations)

	// canvas.yaml and console.yaml are persisted in the database, while the
	// remaining files are committed to git, so the two stores cannot share a
	// single transaction. Commit the git files first: if the git commit fails
	// (for example on a stale head SHA), the request returns before any spec
	// change is written, keeping the database consistent with the failed commit.
	var commitSha string
	if len(gitOps) > 0 {
		commitSha, err = commitGitFileOperations(ctx, gitProvider, orgID, canvasID, organizationID, userID, expectedHeadSha, message, gitOps)
		if err != nil {
			return nil, err
		}
	}

	if len(specOps) > 0 {
		if err := ApplyRepositorySpecFileOperations(
			ctx,
			usageService,
			encryptor,
			registry,
			organizationID,
			id,
			versionID,
			webhookBaseURL,
			authService,
			resolvedAutoLayout,
			specOps,
		); err != nil {
			return nil, err
		}
	}

	return &pb.CommitCanvasRepositoryFilesResponse{
		CommitSha: commitSha,
	}, nil
}

func commitGitFileOperations(
	ctx context.Context,
	gitProvider git.Provider,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	organizationID string,
	userID string,
	expectedHeadSha string,
	message string,
	gitOps []*pb.CanvasRepositoryFileOperation,
) (string, error) {
	repository, err := models.FindRepository(orgID, canvasID)
	if err != nil {
		return "", status.Errorf(codes.NotFound, "repository not found: %v", err)
	}

	user, err := models.FindActiveUserByID(organizationID, userID)
	if err != nil {
		return "", status.Errorf(codes.Internal, "failed to find user: %v", err)
	}

	gitOperations := make([]git.FileOperation, 0, len(gitOps))
	for _, operation := range gitOps {
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
		return "", status.Errorf(codes.Internal, "failed to commit repository files: %v", err)
	}

	return newCommitSha, nil
}
