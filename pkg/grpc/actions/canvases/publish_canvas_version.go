package canvases

import (
	"context"
	"errors"
	"strings"
	"time"

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

	// Validate the name precondition synchronously, before mutating git: the live
	// materialization is deferred to the worker, so a duplicate name would
	// otherwise only surface asynchronously as a materialization error.
	if nameErr := ensureCanvasNameAvailableInTransaction(database.Conn(), organizationUUID, canvasUUID, version.Name); nameErr != nil {
		if errors.Is(nameErr, models.ErrCanvasNameAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, canvasNameAlreadyExistsMessage)
		}
		return nil, status.Errorf(codes.Internal, "failed to check canvas name availability: %v", nameErr)
	}

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

	if err := gitProv.DeleteBranch(ctx, repository.RepoID, draftBranch); err != nil && !errors.Is(err, gitprovider.ErrInvalidRef) {
		return nil, status.Errorf(codes.Internal, "failed to delete draft branch after publish: %v", err)
	}

	var publishedVersion *models.CanvasVersion
	var removed []string
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		// Register a pending published version at the merge commit. It is NOT
		// promoted to live here: the worker's live materialization diffs the
		// current (old) live version against the git snapshot to reconcile
		// workflow_nodes, so the live pointer must not move until then. The worker
		// later reuses this row (matched by commit SHA) and promotes it.
		now := time.Now()
		publishedVersion = &models.CanvasVersion{
			WorkflowID:              canvasUUID,
			State:                   models.CanvasVersionStatePublished,
			Name:                    version.Name,
			Description:             version.Description,
			ChangeManagementEnabled: version.ChangeManagementEnabled,
			ChangeRequestApprovers:  version.ChangeRequestApprovers,
			Nodes:                   version.Nodes,
			Edges:                   version.Edges,
			ConsolePanels:           version.ConsolePanels,
			ConsoleLayout:           version.ConsoleLayout,
			CommitSHA:               mergeSHA,
			GitBranch:               models.CanvasGitBranchMain,
			MaterializationStatus:   models.MaterializationStatusPending,
			PublishedAt:             &now,
			CreatedAt:               &now,
			UpdatedAt:               &now,
		}
		if upsertErr := models.UpsertMaterializedVersionInTransaction(tx, publishedVersion); upsertErr != nil {
			return upsertErr
		}

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
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		if errors.Is(err, models.ErrCanvasNameAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, canvasNameAlreadyExistsMessage)
		}
		return nil, status.Errorf(codes.Internal, "failed to publish canvas: %v", err)
	}
	materialize.PublishDraftBranchDeletionEvents(canvasUUID.String(), removed)

	// Worker-authoritative materialization: main now points at the merge commit,
	// so the worker (inline in tests) materializes the live projection and promotes
	// the published version above.
	if err := materialize.RequestBranchMaterialization(ctx, canvasUUID, models.CanvasGitBranchMain, mergeSHA, &userUUID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to request live materialization: %v", err)
	}

	// Optimistic response: returns the promoted draft content as the published
	// version; the live projection (workflow_nodes, live pointer) is reconciled
	// asynchronously by the worker.
	return publishedVersion, nil
}
