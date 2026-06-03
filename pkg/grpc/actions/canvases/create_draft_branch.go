package canvases

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func CreateDraftBranch(
	ctx context.Context,
	gitProvider git.Provider,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	displayName string,
) (*pb.CreateDraftBranchResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}
	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	repository, err := models.FindRepository(orgUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "repository not found: %v", err)
	}
	if repository.Status != models.RepositoryStatusReady {
		return nil, status.Error(codes.FailedPrecondition, "repository is not ready")
	}

	userUUID := uuid.MustParse(userID)
	branchName, err := materialize.UniqueDraftBranchName(ctx, gitProvider, repository.RepoID, userUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate draft branch name: %v", err)
	}

	createdGitBranch := false
	if !materialize.GitBranchExists(ctx, gitProvider, repository.RepoID, branchName) {
		if err := gitProvider.CreateBranch(ctx, repository.RepoID, branchName, models.CanvasGitBranchMain); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create draft branch: %v", err)
		}
		createdGitBranch = true
	}

	var branch *models.CanvasDraftBranch
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var syncErr error
		branch, syncErr = materialize.SyncDraftBranchFromGit(
			ctx,
			tx,
			gitProvider,
			registry,
			orgUUID,
			canvasUUID,
			branchName,
			materialize.SyncDraftBranchOptions{
				CreatedBy:           &userUUID,
				DisplayNameOverride: strings.TrimSpace(displayName),
			},
		)
		return syncErr
	})
	if err != nil {
		if createdGitBranch {
			_ = gitProvider.DeleteBranch(ctx, repository.RepoID, branchName)
		}
		return nil, status.Errorf(codes.Internal, "failed to create draft branch: %v", err)
	}

	return &pb.CreateDraftBranchResponse{
		Branch: serializeDraftBranch(branch, organizationID, nil),
	}, nil
}
