package materialize

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type syncLiveFromGitOptions struct {
	HeadSHA string
}

// syncLiveFromGit materializes the main branch tip from git into the live DB
// projection. It reads the head and loads the repo snapshot from git before
// opening the database transaction, so no git RPC is held across a pooled DB
// connection. Safe to call repeatedly; git main must already point at the target
// commit.
func syncLiveFromGit(
	ctx context.Context,
	gitProvider git.Provider,
	reg *registry.Registry,
	encryptor crypto.Encryptor,
	authService authorization.Authorization,
	webhookBaseURL string,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	opts syncLiveFromGitOptions,
) (*models.CanvasVersion, error) {
	if gitProvider == nil {
		return nil, fmt.Errorf("git provider is not configured")
	}

	repository, err := models.FindRepositoryUnscoped(canvasID)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %w", err)
	}

	currentHead, err := gitProvider.Head(ctx, repository.RepoID, models.CanvasGitBranchMain)
	if err != nil {
		return nil, fmt.Errorf("failed to read main branch head: %w", err)
	}

	headSHA := strings.TrimSpace(opts.HeadSHA)
	if headSHA == "" {
		headSHA = currentHead
	} else if headSHA != currentHead {
		// Stale notification: a newer commit already superseded this SHA on
		// main, so projecting it would publish an outdated canvas. Skip as a
		// no-op; the commit that is now main's HEAD carries its own
		// notification and will materialize the current state.
		log.Infof(
			"skipping stale live materialization for canvas %s: notification sha %s is not main head %s",
			canvasID, headSHA, currentHead,
		)
		return nil, nil
	}

	snapshot, loadErr := loadRepoSnapshot(ctx, gitProvider, reg, orgID, repository.RepoID, headSHA)
	if loadErr != nil {
		persistLiveMaterializationError(canvasID, headSHA, loadErr)
		publishMainBranchUpdated(canvasID.String(), headSHA, models.MaterializationStatusError, loadErr.Error())
		return nil, loadErr
	}

	live := &liveMaterializer{
		GitProvider:    gitProvider,
		Registry:       reg,
		Encryptor:      encryptor,
		AuthService:    authService,
		WebhookBaseURL: webhookBaseURL,
	}

	var version *models.CanvasVersion
	idempotent := false
	txErr := database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := lockBranchMaterialization(tx, canvasID, models.CanvasGitBranchMain); err != nil {
			return err
		}

		canvas, err := models.FindCanvasInTransaction(tx, orgID, canvasID)
		if err != nil {
			return err
		}

		if existing, done, idempotentErr := syncLiveAlreadyMaterialized(tx, canvasID, canvas, headSHA); idempotentErr != nil {
			return idempotentErr
		} else if done {
			version = existing
			idempotent = true
			return nil
		}

		if nameErr := ensureCanvasNameAvailableInTransaction(tx, orgID, canvasID, snapshot.Name); nameErr != nil {
			return nameErr
		}

		v, matErr := live.materializeLive(ctx, tx, orgID, canvasID, snapshot, headSHA)
		if matErr != nil {
			return matErr
		}
		version = v
		return nil
	})
	if txErr != nil {
		persistLiveMaterializationError(canvasID, headSHA, txErr)
		publishMainBranchUpdated(canvasID.String(), headSHA, models.MaterializationStatusError, txErr.Error())
		return nil, txErr
	}

	if idempotent {
		return version, nil
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
	if canvas.LiveVersionID == nil {
		return nil, false, nil
	}

	liveVersion, err := models.FindCanvasVersionInTransaction(tx, canvasID, *canvas.LiveVersionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if liveVersion.CommitSHA != headSHA ||
		liveVersion.MaterializationStatus != models.MaterializationStatusReady {
		return nil, false, nil
	}

	return liveVersion, true, nil
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
		protoMaterializationStatus(materializationStatus),
		materializationError,
		"",
	).PublishBranchUpdated(); publishErr != nil {
		log.Errorf("failed to publish repository branch updated for main: %v", publishErr)
	}
}

func persistLiveMaterializationError(canvasID uuid.UUID, headSHA string, cause error) {
	if err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return markMaterializationError(tx, canvasID, models.CanvasGitBranchMain, headSHA, nil, cause)
	}); err != nil {
		log.Errorf("failed to persist live materialization error for canvas %s: %v", canvasID, err)
	}
}
