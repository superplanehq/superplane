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

type liveMaterializer struct {
	GitProvider    git.Provider
	Registry       *registry.Registry
	Encryptor      crypto.Encryptor
	AuthService    authorization.Authorization
	WebhookBaseURL string
}

// materializeLive writes the live projection from a snapshot that the caller has
// already loaded from git outside of any transaction, so no git RPC is held
// across the DB connection. The publisher it invokes still runs inside the
// transaction because node Setup() (webhooks, secrets) must commit atomically
// with the node rows.
func (m *liveMaterializer) materializeLive(
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

	if publishErr := messages.NewCanvasUpdatedMessage(canvasID.String(), orgID.String()).PublishUpdated(); publishErr != nil {
		log.Errorf("failed to publish canvas updated message: %v", publishErr)
	}
	if publishErr := messages.NewCanvasVersionUpdatedMessage(canvasID.String(), nextVersion.ID.String()).PublishVersionUpdated(); publishErr != nil {
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
