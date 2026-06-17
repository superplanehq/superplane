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

// MaterializeFromGit projects a branch tip into the database. The underlying sync
// functions load git state before opening their own short transaction, so no git
// RPC is held across a pooled DB connection. It is idempotent: the sync functions
// skip work when the branch is already materialized at commitSHA.
func (m *Materializer) MaterializeFromGit(
	ctx context.Context,
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

	switch mode {
	case ModeLive:
		return SyncLiveFromGit(ctx, m.GitProvider, m.Registry, m.Encryptor, m.AuthService, m.WebhookBaseURL, orgID, canvasID, SyncLiveFromGitOptions{
			HeadSHA: commitSHA,
		})
	default:
		return SyncDraftBranchFromGit(ctx, m.GitProvider, m.Registry, orgID, canvasID, branch, SyncDraftBranchOptions{
			HeadSHA:   commitSHA,
			CreatedBy: ownerID,
		})
	}
}
