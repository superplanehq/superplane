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

func CreateCanvasVersion(
	ctx context.Context,
	gitProvider git.Provider,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	displayName string,
) (*pb.CreateCanvasVersionResponse, error) {
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

	_, err = models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
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

	headSHA, err := gitProvider.Head(ctx, repository.RepoID, branchName)
	if err != nil {
		if createdGitBranch {
			_ = gitProvider.DeleteBranch(ctx, repository.RepoID, branchName)
		}
		return nil, status.Errorf(codes.Internal, "failed to read draft branch head: %v", err)
	}

	var version *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var registerErr error
		version, registerErr = materialize.RegisterPendingDraftVersion(
			tx,
			canvasUUID,
			branchName,
			headSHA,
			materialize.RegisterPendingDraftOptions{
				CreatedBy:           &userUUID,
				DisplayNameOverride: strings.TrimSpace(displayName),
			},
		)
		return registerErr
	})
	if err != nil {
		if createdGitBranch {
			_ = gitProvider.DeleteBranch(ctx, repository.RepoID, branchName)
		}
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to create canvas version: %v", err)
	}

	// Worker-authoritative materialization: git is the source of truth and the
	// pending row above is now committed, so hand off the snapshot load to the
	// materializer worker (run inline in tests) instead of blocking the request.
	if err := materialize.RequestBranchMaterialization(ctx, canvasUUID, branchName, headSHA, &userUUID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to request draft materialization: %v", err)
	}

	// Optimistic response: the draft branch is registered in the "pending"
	// materialization state; nodes/edges are filled in asynchronously by the
	// worker, so only metadata is guaranteed here.
	return &pb.CreateCanvasVersionResponse{
		Version: SerializeCanvasVersion(version, organizationID, nil),
	}, nil
}
