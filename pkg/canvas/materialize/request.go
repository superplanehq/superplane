package materialize

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

// BranchMaterializer projects the tip of a git branch into the database: the live
// version plus workflow_nodes/webhooks for main, or the draft version for draft
// branches. It is the shared core used both by the RepositoryMaterializerWorker
// (driven by RabbitMQ) and by the in-process synchronous materializer used in
// tests, so the materialization logic lives in exactly one place.
type BranchMaterializer struct {
	GitProvider    git.Provider
	Registry       *registry.Registry
	Encryptor      crypto.Encryptor
	AuthService    authorization.Authorization
	WebhookBaseURL string
}

// inProcessMaterializer, when set, makes RequestBranchMaterialization run
// materialization synchronously in-process instead of publishing a
// repository_branch_updated message for the worker to consume. Production never
// sets this: handlers stay worker-authoritative and the worker does the work.
// Tests wire it (via support.Setup) so that materialized state is observable
// without running the RabbitMQ worker, while the request path still goes through
// the exact same "register pending row + request materialization" flow.
var (
	inProcessMu           sync.RWMutex
	inProcessMaterializer *BranchMaterializer
)

// SetInProcessMaterializer installs (or clears, with nil) the synchronous
// in-process materializer. Test-only; never called from production code.
func SetInProcessMaterializer(m *BranchMaterializer) {
	inProcessMu.Lock()
	defer inProcessMu.Unlock()
	inProcessMaterializer = m
}

// RequestBranchMaterialization asks the materializer worker to project the tip of
// branch into the database. Handlers call this after committing to git and
// registering a pending workflow_versions row, so the expensive snapshot load and
// node/webhook reconciliation happens in the worker instead of on the request
// path. In tests an in-process materializer runs the same work synchronously.
func RequestBranchMaterialization(ctx context.Context, canvasID uuid.UUID, branch, headSHA string, pushedBy *uuid.UUID) error {
	inProcessMu.RLock()
	m := inProcessMaterializer
	inProcessMu.RUnlock()
	if m != nil {
		return m.MaterializeBranch(ctx, canvasID, branch, headSHA, pushedBy)
	}

	pushedByID := ""
	if pushedBy != nil {
		pushedByID = pushedBy.String()
	}

	return messages.NewRepositoryBranchUpdatedMessage(
		canvasID.String(),
		branch,
		headSHA,
		protoMaterializationStatus(models.MaterializationStatusPending),
		"",
		pushedByID,
	).PublishBranchUpdated()
}

// MaterializeBranch reconciles draft-branch deletions and then materializes the
// branch tip (live for main, draft otherwise). It is idempotent: the underlying
// sync functions skip work when the branch is already materialized at headSHA.
func (m *BranchMaterializer) MaterializeBranch(ctx context.Context, canvasID uuid.UUID, branch, headSHA string, pushedBy *uuid.UUID) error {
	canvas, err := models.FindCanvasWithoutOrgScope(canvasID)
	if err != nil {
		return err
	}

	var removed []string
	if err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var reconcileErr error
		removed, reconcileErr = ReconcileDraftBranchDeletionsFromGit(ctx, tx, m.GitProvider, canvasID, ReconcileDraftBranchDeletionsOptions{})
		return reconcileErr
	}); err != nil {
		return err
	}
	PublishDraftBranchDeletionEvents(canvasID.String(), removed)

	if headSHA == "" {
		repository, repoErr := models.FindRepositoryUnscoped(canvasID)
		if repoErr != nil {
			return repoErr
		}
		headSHA, err = m.GitProvider.Head(ctx, repository.RepoID, branch)
		if err != nil {
			return err
		}
	}

	if branch == models.CanvasGitBranchMain {
		if m.liveAlreadyMaterialized(canvasID, headSHA) {
			return nil
		}

		return database.Conn().Transaction(func(tx *gorm.DB) error {
			_, syncErr := SyncLiveFromGit(
				ctx,
				tx,
				m.GitProvider,
				m.Registry,
				m.Encryptor,
				m.AuthService,
				m.WebhookBaseURL,
				canvas.OrganizationID,
				canvasID,
				// Git main is authoritative: by the time a commit lands on main it
				// has already passed change-management gating at the write path
				// (publish/change-request handlers), so the materializer always
				// projects what main points at.
				SyncLiveFromGitOptions{HeadSHA: headSHA},
			)
			return syncErr
		})
	}

	if m.draftAlreadyMaterialized(canvasID, branch, headSHA) {
		return nil
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		_, syncErr := SyncDraftBranchFromGit(
			ctx,
			tx,
			m.GitProvider,
			m.Registry,
			canvas.OrganizationID,
			canvasID,
			branch,
			SyncDraftBranchOptions{HeadSHA: headSHA, CreatedBy: pushedBy},
		)
		return syncErr
	})
}

func (m *BranchMaterializer) liveAlreadyMaterialized(canvasID uuid.UUID, headSHA string) bool {
	liveVersion, err := models.FindLiveCanvasVersion(canvasID)
	if err != nil {
		return false
	}

	return liveVersion.CommitSHA == headSHA &&
		liveVersion.MaterializationStatus == models.MaterializationStatusReady
}

func (m *BranchMaterializer) draftAlreadyMaterialized(canvasID uuid.UUID, branch, headSHA string) bool {
	draftVersion, err := models.FindDraftVersionByBranch(canvasID, branch)
	return err == nil && draftVersion != nil &&
		draftVersion.CommitSHA == headSHA &&
		draftVersion.MaterializationStatus == models.MaterializationStatusReady
}
