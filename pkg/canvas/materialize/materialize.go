package materialize

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type Mode int

const (
	ModeDraft Mode = iota
	ModeLive
)

type Materializer struct {
	GitProvider    git.Provider
	Registry       *registry.Registry
	Encryptor      crypto.Encryptor
	AuthService    authorization.Authorization
	WebhookBaseURL string
}

func (m *Materializer) MaterializeFromGit(
	ctx context.Context,
	tx *gorm.DB,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	branch string,
	commitSHA string,
	mode Mode,
	ownerID *uuid.UUID,
) (*models.CanvasVersion, error) {
	if m == nil {
		return nil, fmt.Errorf("materializer is not configured")
	}

	state, err := models.FindRepositoryMaterializationStateInTransaction(tx, canvasID, branch)
	if err == nil &&
		state.MaterializedSHA == commitSHA &&
		state.Status == models.MaterializationStatusReady {
		version, findErr := models.FindVersionByCommitSHAInTransaction(tx, canvasID, commitSHA)
		if findErr == nil && version.MaterializationStatus == models.MaterializationStatusReady {
			return version, nil
		}
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	switch mode {
	case ModeLive:
		return SyncLiveFromGit(ctx, tx, m.GitProvider, m.Registry, m.Encryptor, m.AuthService, m.WebhookBaseURL, orgID, canvasID, SyncLiveFromGitOptions{
			HeadSHA: commitSHA,
		})
	default:
		draft := &DraftMaterializer{
			GitProvider: m.GitProvider,
			Registry:    m.Registry,
		}
		return draft.MaterializeDraft(ctx, tx, orgID, canvasID, branch, commitSHA, ownerID)
	}
}
