package canvases

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteCanvasVersion(
	ctx context.Context,
	gitProvider git.Provider,
	organizationID string,
	canvasID string,
	versionID string,
) (*pb.DeleteCanvasVersionResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	versionUUID, err := uuid.Parse(versionID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version id: %v", err)
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	userUUID := uuid.MustParse(userID)
	version, err := models.FindCanvasVersion(canvasUUID, versionUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "version not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load version: %v", err)
	}

	if version.State != models.CanvasVersionStateDraft {
		return nil, status.Error(codes.FailedPrecondition, "only draft versions can be discarded")
	}

	if !models.IsRegisteredDraftVersion(version) {
		return nil, status.Error(codes.FailedPrecondition, "version is not a registered draft branch")
	}

	if version.OwnerID == nil || *version.OwnerID != userUUID {
		return nil, status.Error(codes.PermissionDenied, "version owner mismatch")
	}

	if version.BranchName == nil || strings.TrimSpace(*version.BranchName) == "" {
		return nil, status.Error(codes.FailedPrecondition, "draft branch is required")
	}

	branchName := strings.TrimSpace(*version.BranchName)
	if branchName == models.CanvasGitBranchMain {
		return nil, status.Error(codes.InvalidArgument, "cannot delete main branch")
	}

	repository, err := models.FindRepository(orgUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "repository not found: %v", err)
	}

	gitExists := materialize.GitBranchExists(ctx, gitProvider, repository.RepoID, branchName)
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

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), versionUUID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas version updated RabbitMQ message: %v", err)
	}

	return &pb.DeleteCanvasVersionResponse{}, nil
}
