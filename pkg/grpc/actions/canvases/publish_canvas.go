package canvases

import (
	"context"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func PublishCanvas(
	ctx context.Context,
	gitProvider git.Provider,
	registry *registry.Registry,
	encryptor crypto.Encryptor,
	authService authorization.Authorization,
	webhookBaseURL string,
	organizationID string,
	canvasID string,
	draftBranch string,
) (*pb.PublishCanvasResponse, error) {
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

	draftBranch = strings.TrimSpace(draftBranch)
	if draftBranch == "" {
		draftBranch = materialize.DefaultDraftBranchName(uuid.MustParse(userID))
	}
	if draftBranch == models.CanvasGitBranchMain {
		return nil, status.Error(codes.InvalidArgument, "draft branch is required")
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}
	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	changeManagementEnabled, modeErr := isChangeManagementEnabledForCanvas(canvas)
	if modeErr != nil {
		return nil, status.Errorf(codes.Internal, "failed to load change management setting: %v", modeErr)
	}
	if changeManagementEnabled {
		return nil, status.Error(codes.FailedPrecondition, "change management is enabled for this canvas; create a change request instead")
	}

	repository, err := models.FindRepository(orgUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "repository not found: %v", err)
	}

	user, err := models.FindActiveUserByID(organizationID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user: %v", err)
	}

	mergeSHA, err := gitProvider.MergeBranch(
		ctx,
		repository.RepoID,
		draftBranch,
		models.CanvasGitBranchMain,
		"Publish canvas",
		git.CommitAuthor{Name: user.Name, Email: user.GetEmail()},
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to merge draft branch: %v", err)
	}

	var publishedVersion *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		nameErr := ensureCanvasNameAvailableInTransaction(tx, orgUUID, canvasUUID, canvas.Name)
		if nameErr != nil {
			return nameErr
		}

		mat := &materialize.Materializer{
			GitProvider:    gitProvider,
			Registry:       registry,
			Encryptor:      encryptor,
			AuthService:    authService,
			WebhookBaseURL: webhookBaseURL,
		}

		version, matErr := mat.MaterializeFromGit(ctx, tx, orgUUID, canvasUUID, models.CanvasGitBranchMain, mergeSHA, materialize.ModeLive, nil)
		if matErr != nil {
			return matErr
		}

		refreshErr := refreshOpenCanvasChangeRequestsInTransaction(tx, orgUUID, canvasUUID, uuid.Nil)
		if refreshErr != nil {
			log.Errorf("failed to refresh open canvas change requests: %v", refreshErr)
			return refreshErr
		}

		if deleteErr := models.DeleteDraftBranchInTransaction(tx, canvasUUID, draftBranch); deleteErr != nil {
			return deleteErr
		}
		if deleteErr := models.DeleteRepositoryMaterializationStateInTransaction(tx, canvasUUID, draftBranch); deleteErr != nil {
			return deleteErr
		}

		publishedVersion = version
		return nil
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to publish canvas: %v", err)
	}

	if err := gitProvider.DeleteBranch(ctx, repository.RepoID, draftBranch); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete draft branch after publish: %v", err)
	}

	return &pb.PublishCanvasResponse{
		Version: SerializeCanvasVersion(publishedVersion, organizationID),
	}, nil
}
