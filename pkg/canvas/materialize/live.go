package materialize

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type LiveMaterializer struct {
	GitProvider    git.Provider
	Registry       *registry.Registry
	Encryptor      crypto.Encryptor
	AuthService    authorization.Authorization
	WebhookBaseURL string
}

func (m *LiveMaterializer) MaterializeLive(
	ctx context.Context,
	tx *gorm.DB,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	commitSHA string,
) (*models.CanvasVersion, error) {
	if m == nil || m.GitProvider == nil {
		return nil, fmt.Errorf("live materializer is not configured")
	}

	existing, err := models.FindVersionBySHAInTransaction(tx, canvasID, commitSHA)
	if err == nil && existing.MaterializationStatus == models.MaterializationStatusReady && existing.State == models.CanvasVersionStatePublished {
		canvas, canvasErr := models.FindCanvasInTransaction(tx, orgID, canvasID)
		if canvasErr != nil {
			return nil, canvasErr
		}
		if canvas.LiveVersionID != nil && *canvas.LiveVersionID == commitSHA {
			return existing, nil
		}
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	repository, err := models.FindRepositoryInTransaction(tx, canvasID)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %w", err)
	}

	snapshot, loadErr := LoadRepoSnapshot(ctx, m.GitProvider, m.Registry, orgID, repository.RepoID, commitSHA)
	if loadErr != nil {
		if markErr := markMaterializationError(tx, canvasID, models.CanvasGitBranchMain, commitSHA, loadErr); markErr != nil {
			return nil, markErr
		}
		return nil, loadErr
	}

	canvas, err := models.FindCanvasInTransaction(tx, orgID, canvasID)
	if err != nil {
		return nil, err
	}

	var liveVersion *models.CanvasVersion
	if canvas.LiveVersionID != nil && *canvas.LiveVersionID != commitSHA {
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
		ID:                      commitSHA,
		WorkflowID:              canvasID,
		State:                   models.CanvasVersionStatePublished,
		Name:                    snapshot.Name,
		Description:             snapshot.Description,
		ChangeManagementEnabled: snapshot.ChangeManagementEnabled,
		ChangeRequestApprovers:  datatypes.NewJSONSlice(snapshot.ChangeRequestApprovers),
		Nodes:                   datatypes.NewJSONSlice(snapshot.Nodes),
		Edges:                   datatypes.NewJSONSlice(snapshot.Edges),
		ConsolePanels:           datatypes.NewJSONType(snapshot.ConsolePanels),
		ConsoleLayout:           datatypes.NewJSONType(snapshot.ConsoleLayout),
		GitBranch:               models.CanvasGitBranchMain,
		MaterializationStatus:   models.MaterializationStatusReady,
		PublishedAt:             &now,
		CreatedAt:               &now,
		UpdatedAt:               &now,
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

	if err := models.UpsertRepositoryMaterializationStateInTransaction(tx, &models.RepositoryMaterializationState{
		CanvasID:        canvasID,
		Branch:          models.CanvasGitBranchMain,
		HeadSHA:         commitSHA,
		MaterializedSHA: commitSHA,
		Status:          models.MaterializationStatusReady,
	}); err != nil {
		return nil, err
	}

	if publishErr := messages.NewCanvasUpdatedMessage(canvasID.String(), orgID.String()).PublishUpdated(); publishErr != nil {
		log.Errorf("failed to publish canvas updated message: %v", publishErr)
	}
	if publishErr := messages.NewCanvasVersionUpdatedMessage(canvasID.String(), commitSHA).PublishVersionUpdated(); publishErr != nil {
		log.Errorf("failed to publish canvas version updated message: %v", publishErr)
	}

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
