package canvases

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
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
	_ = usageService
	_ = encryptor
	_ = webhookBaseURL
	_ = authService
	_ = autoLayout

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

	repository, err := models.FindRepository(orgID, canvasID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "repository not found: %v", err)
	}

	user, err := models.FindActiveUserByID(organizationID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user: %v", err)
	}

	branch := models.CanvasGitBranchMain
	var ownerID *uuid.UUID
	if strings.TrimSpace(versionID) != "" {
		versionUUID, parseErr := uuid.Parse(versionID)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid version id: %v", parseErr)
		}

		version, findErr := models.FindCanvasVersion(canvasID, versionUUID)
		if findErr != nil {
			return nil, status.Errorf(codes.NotFound, "version not found: %v", findErr)
		}
		if version.BranchName == nil || strings.TrimSpace(*version.BranchName) == "" {
			return nil, status.Error(codes.FailedPrecondition, "draft branch is required")
		}
		branch = strings.TrimSpace(*version.BranchName)
		ownerID = version.OwnerID
	} else {
		branch = materialize.DefaultDraftBranchName(uuid.MustParse(userID))
	}

	if branch == models.CanvasGitBranchMain {
		return nil, status.Error(codes.InvalidArgument, "commits must target a draft branch")
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

	if len(gitOperations) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one file operation is required")
	}

	if strings.TrimSpace(message) == "" {
		message = "Update repository files"
	}

	newCommitSha, err := gitProvider.Commit(ctx, repository.RepoID, git.CommitOptions{
		Branch:          branch,
		BaseBranch:      branch,
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

	userUUID := uuid.MustParse(userID)
	if ownerID == nil {
		ownerID = &userUUID
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		mat := &materialize.DraftMaterializer{GitProvider: gitProvider, Registry: registry}
		materialized, matErr := mat.MaterializeDraft(ctx, tx, orgID, canvasID, branch, newCommitSha, ownerID)
		if matErr != nil {
			return matErr
		}

		// A direct commit re-materializes the draft from git, so any staged DB
		// edits for this version are now stale and must be cleared.
		if materialized != nil {
			return models.DiscardWorkflowStagingInTransaction(tx, materialized.ID, nil)
		}
		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to materialize draft: %v", err)
	}

	return &pb.CommitCanvasRepositoryFilesResponse{
		CommitSha: newCommitSha,
	}, nil
}
