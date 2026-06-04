package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteDraftBranch(
	ctx context.Context,
	gitProvider git.Provider,
	organizationID string,
	canvasID string,
	branchName string,
) (*pb.DeleteDraftBranchResponse, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	if branchName == "" || branchName == models.CanvasGitBranchMain {
		return nil, status.Error(codes.InvalidArgument, "cannot delete main branch")
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

	gitExists := materialize.GitBranchExists(ctx, gitProvider, repository.RepoID, branchName)
	_, dbErr := models.FindDraftBranch(canvasUUID, branchName)
	if dbErr != nil && !errors.Is(dbErr, gorm.ErrRecordNotFound) {
		return nil, status.Errorf(codes.Internal, "failed to find draft branch: %v", dbErr)
	}
	dbExists := dbErr == nil
	if !gitExists && !dbExists {
		return nil, status.Error(codes.NotFound, "draft branch not found")
	}

	if gitExists {
		if err := gitProvider.DeleteBranch(ctx, repository.RepoID, branchName); err != nil && !errors.Is(err, git.ErrInvalidRef) {
			return nil, status.Errorf(codes.Internal, "failed to delete git branch: %v", err)
		}
	}

	var removed []string
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var reconcileErr error
		removed, reconcileErr = materialize.ReconcileDraftBranchDeletionsFromGit(
			ctx,
			tx,
			gitProvider,
			canvasUUID,
			materialize.ReconcileDraftBranchDeletionsOptions{BranchName: branchName},
		)
		return reconcileErr
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete draft branch: %v", err)
	}

	materialize.PublishDraftBranchDeletionEvents(canvasID, removed)

	return &pb.DeleteDraftBranchResponse{}, nil
}
