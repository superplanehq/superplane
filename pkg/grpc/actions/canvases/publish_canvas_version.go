package canvases

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func PublishCanvasVersion(
	ctx context.Context,
	encryptor crypto.Encryptor,
	reg *registry.Registry,
	gitProv gitprovider.Provider,
	organizationID string,
	canvasID string,
	versionID string,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*pb.PublishCanvasVersionResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	versionUUID, err := uuid.Parse(versionID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version id: %v", err)
	}

	organizationUUID := uuid.MustParse(organizationID)
	userUUID := uuid.MustParse(userID)

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	changeManagementEnabled, modeErr := isChangeManagementEnabledForCanvas(canvas)
	if modeErr != nil {
		return nil, status.Errorf(codes.Internal, "failed to load change management setting: %v", modeErr)
	}
	if changeManagementEnabled {
		return nil, status.Error(codes.FailedPrecondition, "change management is enabled for this canvas; create a change request instead")
	}

	publishedVersion, err := publishDraftVersionInTransaction(
		ctx, encryptor, reg, gitProv, organizationID, organizationUUID, canvasUUID, versionUUID, userUUID, authService, webhookBaseURL,
	)
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, actions.ToStatus(err)
	}

	if err := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}
	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), publishedVersion.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas version updated RabbitMQ message: %v", err)
	}

	return &pb.PublishCanvasVersionResponse{
		Version: SerializeCanvasVersion(publishedVersion, organizationID),
	}, nil
}

func publishDraftVersionInTransaction(
	ctx context.Context,
	encryptor crypto.Encryptor,
	reg *registry.Registry,
	gitProv gitprovider.Provider,
	organizationID string,
	organizationUUID uuid.UUID,
	canvasUUID uuid.UUID,
	versionUUID uuid.UUID,
	userUUID uuid.UUID,
	authService authorization.Authorization,
	webhookBaseURL string,
) (*models.CanvasVersion, error) {
	version, err := models.FindCanvasVersion(canvasUUID, versionUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "version not found")
		}
		return nil, err
	}

	if version.State != models.CanvasVersionStateDraft {
		return nil, status.Error(codes.FailedPrecondition, "only draft versions can be published")
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

	draftBranch := strings.TrimSpace(*version.BranchName)

	repository, err := models.FindRepository(organizationUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "repository not found: %v", err)
	}

	user, err := models.FindActiveUserByID(organizationID, userUUID.String())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user: %v", err)
	}

	mergeSHA, err := gitProv.MergeBranch(
		ctx,
		repository.RepoID,
		draftBranch,
		models.CanvasGitBranchMain,
		"Publish canvas",
		gitprovider.CommitAuthor{Name: user.Name, Email: user.GetEmail()},
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to merge draft branch: %v", err)
	}

	var publishedVersion *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		synced, syncErr := materialize.SyncLiveFromGit(
			ctx,
			tx,
			gitProv,
			reg,
			encryptor,
			authService,
			webhookBaseURL,
			organizationUUID,
			canvasUUID,
			materialize.SyncLiveFromGitOptions{
				HeadSHA:                   mergeSHA,
				SkipChangeManagementCheck: true,
			},
		)
		if syncErr != nil {
			return syncErr
		}

		publishedVersion = synced
		return nil
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		if errors.Is(err, models.ErrCanvasNameAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, canvasNameAlreadyExistsMessage)
		}
		return nil, status.Errorf(codes.Internal, "failed to publish canvas: %v", err)
	}

	if err := gitProv.DeleteBranch(ctx, repository.RepoID, draftBranch); err != nil && !errors.Is(err, gitprovider.ErrInvalidRef) {
		return nil, status.Errorf(codes.Internal, "failed to delete draft branch after publish: %v", err)
	}

	var removed []string
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var reconcileErr error
		removed, reconcileErr = materialize.ReconcileDraftBranchDeletionsFromGit(
			ctx,
			tx,
			gitProv,
			canvasUUID,
			materialize.ReconcileDraftBranchDeletionsOptions{BranchName: draftBranch},
		)
		return reconcileErr
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to reconcile draft branch deletion after publish: %v", err)
	}
	materialize.PublishDraftBranchDeletionEvents(canvasUUID.String(), removed)

	return publishedVersion, nil
}
