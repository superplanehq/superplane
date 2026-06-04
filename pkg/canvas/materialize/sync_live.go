package materialize

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/canvas/changerequests"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type SyncLiveFromGitOptions struct {
	HeadSHA                   string
	SkipChangeManagementCheck bool
}

// SyncLiveFromGit materializes the main branch tip from git into the live DB projection.
// Safe to call repeatedly; git main must already point at the target commit.
func SyncLiveFromGit(
	ctx context.Context,
	tx *gorm.DB,
	gitProvider git.Provider,
	reg *registry.Registry,
	encryptor crypto.Encryptor,
	authService authorization.Authorization,
	webhookBaseURL string,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	opts SyncLiveFromGitOptions,
) (*models.CanvasVersion, error) {
	if gitProvider == nil {
		return nil, fmt.Errorf("git provider is not configured")
	}

	repository, err := models.FindRepositoryInTransaction(tx, canvasID)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %w", err)
	}

	headSHA := strings.TrimSpace(opts.HeadSHA)
	if headSHA == "" {
		headSHA, err = gitProvider.Head(ctx, repository.RepoID, models.CanvasGitBranchMain)
		if err != nil {
			return nil, fmt.Errorf("failed to read main branch head: %w", err)
		}
	}

	canvas, err := models.FindCanvasInTransaction(tx, orgID, canvasID)
	if err != nil {
		return nil, err
	}

	if !opts.SkipChangeManagementCheck {
		changeManagementEnabled, cmErr := isChangeManagementEnabledInTransaction(tx, canvas)
		if cmErr != nil {
			return nil, cmErr
		}
		if changeManagementEnabled {
			cmErr := status.Error(codes.FailedPrecondition, "change management is enabled for this canvas; create a change request instead")
			persistLiveMaterializationError(canvasID, headSHA, cmErr)
			publishMainBranchUpdated(canvasID.String(), headSHA, models.MaterializationStatusError, cmErr.Error())
			return nil, cmErr
		}
	}

	if version, done, idempotentErr := syncLiveAlreadyMaterialized(tx, canvasID, canvas, headSHA); idempotentErr != nil {
		return nil, idempotentErr
	} else if done {
		return version, nil
	}

	snapshot, loadErr := LoadRepoSnapshot(ctx, gitProvider, reg, orgID, repository.RepoID, headSHA)
	if loadErr != nil {
		if markErr := markMaterializationError(tx, canvasID, models.CanvasGitBranchMain, headSHA, loadErr); markErr != nil {
			return nil, markErr
		}
		publishMainBranchUpdated(canvasID.String(), headSHA, models.MaterializationStatusError, loadErr.Error())
		return nil, loadErr
	}

	if nameErr := ensureCanvasNameAvailableInTransaction(tx, orgID, canvasID, snapshot.Name); nameErr != nil {
		if markErr := markMaterializationError(tx, canvasID, models.CanvasGitBranchMain, headSHA, nameErr); markErr != nil {
			return nil, markErr
		}
		publishMainBranchUpdated(canvasID.String(), headSHA, models.MaterializationStatusError, nameErr.Error())
		return nil, nameErr
	}

	live := &LiveMaterializer{
		GitProvider:    gitProvider,
		Registry:       reg,
		Encryptor:      encryptor,
		AuthService:    authService,
		WebhookBaseURL: webhookBaseURL,
	}
	version, matErr := live.MaterializeLive(ctx, tx, orgID, canvasID, headSHA)
	if matErr != nil {
		publishMainBranchUpdated(canvasID.String(), headSHA, models.MaterializationStatusError, matErr.Error())
		return nil, matErr
	}

	if refreshErr := changerequests.RefreshOpenCanvasChangeRequestsInTransaction(tx, orgID, canvasID, uuid.Nil); refreshErr != nil {
		return nil, refreshErr
	}

	publishMainBranchUpdated(canvasID.String(), headSHA, models.MaterializationStatusReady, "")
	return version, nil
}

func syncLiveAlreadyMaterialized(
	tx *gorm.DB,
	canvasID uuid.UUID,
	canvas *models.Canvas,
	headSHA string,
) (*models.CanvasVersion, bool, error) {
	state, err := models.FindRepositoryMaterializationStateInTransaction(tx, canvasID, models.CanvasGitBranchMain)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	if err != nil ||
		state.MaterializedSHA != headSHA ||
		state.Status != models.MaterializationStatusReady ||
		canvas.LiveVersionID == nil ||
		*canvas.LiveVersionID != headSHA {
		return nil, false, nil
	}

	version, findErr := models.FindVersionBySHAInTransaction(tx, canvasID, headSHA)
	if findErr != nil {
		return nil, false, findErr
	}
	if version.MaterializationStatus != models.MaterializationStatusReady {
		return nil, false, nil
	}

	return version, true, nil
}

func isChangeManagementEnabledInTransaction(tx *gorm.DB, canvas *models.Canvas) (bool, error) {
	if canvas == nil || canvas.IsTemplate {
		return false, nil
	}

	organizationChangeManagementEnabled, err := models.IsChangeManagementEnabledInTransaction(tx, canvas.OrganizationID)
	if err != nil {
		return false, err
	}
	if organizationChangeManagementEnabled {
		return true, nil
	}

	return canvas.ChangeManagementEnabled, nil
}

func ensureCanvasNameAvailableInTransaction(
	tx *gorm.DB,
	organizationID uuid.UUID,
	canvasID uuid.UUID,
	name string,
) error {
	existingCanvas, err := models.FindCanvasByNameInTransaction(tx, name, organizationID)
	if err == nil && existingCanvas.ID != canvasID {
		return models.ErrCanvasNameAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return nil
}

func publishMainBranchUpdated(canvasID, headSHA, materializationStatus, materializationError string) {
	if publishErr := messages.NewRepositoryBranchUpdatedMessage(
		canvasID,
		models.CanvasGitBranchMain,
		headSHA,
		materializationStatus,
		materializationError,
		"",
	).PublishBranchUpdated(); publishErr != nil {
		log.Errorf("failed to publish repository branch updated for main: %v", publishErr)
	}
}

func persistLiveMaterializationError(canvasID uuid.UUID, headSHA string, cause error) {
	if err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return markMaterializationError(tx, canvasID, models.CanvasGitBranchMain, headSHA, cause)
	}); err != nil {
		log.Errorf("failed to persist live materialization error for canvas %s: %v", canvasID, err)
	}
}
