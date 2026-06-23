package materialize

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type liveMaterializer struct {
	GitProvider    git.Provider
	Registry       *registry.Registry
	Encryptor      crypto.Encryptor
	AuthService    authorization.Authorization
	WebhookBaseURL string
}

// persist runs the live materialization inside a single transaction: it holds the
// branch lock, runs the authoritative idempotency check, ensures the canvas name
// is available, creates a new CanvasVersion instance, upserts the workflow_versions row,
// and publishes the canvas.
//
// if the live version already matched headSHA it returns the existing version
// without doing any work.
func (m *liveMaterializer) persist(
	ctx context.Context,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	headSHA string,
	snapshot *repoSnapshot,
) (*models.CanvasVersion, error) {
	var version *models.CanvasVersion
	txErr := database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := lockBranchMaterialization(tx, canvasID, models.CanvasGitBranchMain); err != nil {
			return err
		}

		canvas, err := models.FindCanvasInTransaction(tx, orgID, canvasID)
		if err != nil {
			return err
		}

		if existing, done, idempotentErr := liveAlreadyMaterializedInTransaction(tx, canvasID, canvas, headSHA); idempotentErr != nil {
			return idempotentErr
		} else if done {
			version = existing
			return nil
		}

		if nameErr := ensureCanvasNameAvailableInTransaction(tx, orgID, canvasID, snapshot.Name); nameErr != nil {
			return nameErr
		}

		v, matErr := m.materializeLiveInTransaction(ctx, tx, orgID, canvasID, snapshot, headSHA)
		if matErr != nil {
			return matErr
		}
		version = v
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return version, nil
}

// liveAlreadyMaterializedInTransaction is the authoritative, lock-protected
// idempotency check for the live (main) branch.
//
// It returns the existing live version (nil when none exists) and a
// boolean flag that is true when that version is already materialized at headSHA.
func liveAlreadyMaterializedInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	canvas *models.Canvas,
	headSHA string,
) (*models.CanvasVersion, bool, error) {
	if canvas == nil || canvas.LiveVersionID == nil {
		return nil, false, nil
	}

	version, err := models.FindCanvasVersionInTransaction(tx, canvasID, *canvas.LiveVersionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return version, materializedAt(version, headSHA), nil
}

func (m *liveMaterializer) materializeLiveInTransaction(
	ctx context.Context,
	tx *gorm.DB,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	snapshot *repoSnapshot,
	commitSHA string,
) (*models.CanvasVersion, error) {
	if m == nil || m.GitProvider == nil {
		return nil, fmt.Errorf("live materializer is not configured")
	}

	existing, err := models.FindVersionByCommitSHAInTransaction(tx, canvasID, commitSHA)
	if err == nil &&
		existing.MaterializationStatus == models.MaterializationStatusReady &&
		existing.State == models.CanvasVersionStatePublished {
		canvas, canvasErr := models.FindCanvasInTransaction(tx, orgID, canvasID)
		if canvasErr != nil {
			return nil, canvasErr
		}
		if canvas.LiveVersionID != nil && *canvas.LiveVersionID == existing.ID {
			return existing, nil
		}
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	canvas, err := models.FindCanvasInTransaction(tx, orgID, canvasID)
	if err != nil {
		return nil, err
	}

	var liveVersion *models.CanvasVersion
	if canvas.LiveVersionID != nil {
		liveVersion, err = models.FindCanvasVersionInTransaction(tx, canvasID, *canvas.LiveVersionID)
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	if liveVersion == nil {
		liveVersion = &models.CanvasVersion{
			WorkflowID: canvasID,
			Nodes:      datatypes.NewJSONSlice([]models.Node{}),
			Edges:      datatypes.NewJSONSlice([]models.Edge{}),
		}
	}

	now := time.Now()
	nextVersion := &models.CanvasVersion{
		WorkflowID:            canvasID,
		State:                 models.CanvasVersionStatePublished,
		Name:                  snapshot.Name,
		Description:           snapshot.Description,
		Nodes:                 datatypes.NewJSONSlice(snapshot.Nodes),
		Edges:                 datatypes.NewJSONSlice(snapshot.Edges),
		ConsolePanels:         datatypes.NewJSONType(snapshot.ConsolePanels),
		ConsoleLayout:         datatypes.NewJSONType(snapshot.ConsoleLayout),
		CommitSHA:             commitSHA,
		GitBranch:             models.CanvasGitBranchMain,
		MaterializationStatus: models.MaterializationStatusReady,
		PublishedAt:           &now,
		CreatedAt:             &now,
		UpdatedAt:             &now,
	}

	if err := models.UpsertMaterializedVersionInTransaction(tx, nextVersion); err != nil {
		return nil, err
	}

	if err := publishLiveVersion(ctx, tx, liveVersion, nextVersion, changesets.CanvasPublisherOptions{
		Registry:       m.Registry,
		OrgID:          orgID,
		Encryptor:      m.Encryptor,
		AuthService:    m.AuthService,
		WebhookBaseURL: m.WebhookBaseURL,
	}); err != nil {
		return nil, err
	}

	publishCanvasUpdated(canvasID.String(), orgID.String())
	publishCanvasVersionUpdated(canvasID.String(), nextVersion.ID.String())

	return nextVersion, nil
}

func publishLiveVersion(
	ctx context.Context,
	tx *gorm.DB,
	liveVersion *models.CanvasVersion,
	nextVersion *models.CanvasVersion,
	options changesets.CanvasPublisherOptions,
) error {
	changeset, err := changesets.NewChangesetBuilder(
		liveVersion.Nodes,
		liveVersion.Edges,
		nextVersion.Nodes,
		nextVersion.Edges,
	).Build()
	if err != nil {
		return err
	}

	if len(changeset.GetChanges()) == 0 {
		return models.PromoteToLiveInTransaction(tx, nextVersion, nextVersion.Nodes, nextVersion.Edges)
	}

	publisher, err := changesets.NewCanvasPublisher(tx, nextVersion, liveVersion, options)
	if err != nil {
		return err
	}

	return publisher.Publish(ctx)
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
