package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
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

	if _, err := models.FindDraftBranch(canvasUUID, branchName); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "draft branch not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to find draft branch: %v", err)
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		if deleteErr := models.DeleteDraftBranchInTransaction(tx, canvasUUID, branchName); deleteErr != nil {
			return deleteErr
		}
		return models.DeleteRepositoryMaterializationStateInTransaction(tx, canvasUUID, branchName)
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete draft branch: %v", err)
	}

	if err := gitProvider.DeleteBranch(ctx, repository.RepoID, branchName); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete git branch: %v", err)
	}

	return &pb.DeleteDraftBranchResponse{}, nil
}
