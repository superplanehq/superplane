package materialize

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
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

// MaterializeBranch reconciles draft-branch deletions and then materializes the
// branch tip (live for main, draft otherwise). It is idempotent: the underlying
// sync functions skip work when the branch is already materialized at headSHA.
func (m *BranchMaterializer) MaterializeBranch(ctx context.Context, canvasID uuid.UUID, branch, headSHA string, pushedBy *uuid.UUID) error {
	canvas, err := models.FindCanvasWithoutOrgScope(canvasID)
	if err != nil {
		return err
	}

	removed, err := reconcileDraftBranchDeletionsFromGit(ctx, m.GitProvider, canvasID, reconcileDraftBranchDeletionsOptions{})
	if err != nil {
		return err
	}
	publishDraftBranchDeletionEvents(canvasID.String(), removed)

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

		_, syncErr := syncLiveFromGit(
			ctx,
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
			syncLiveFromGitOptions{HeadSHA: headSHA},
		)
		return syncErr
	}

	if m.draftAlreadyMaterialized(canvasID, branch, headSHA) {
		return nil
	}

	_, syncErr := syncDraftBranchFromGit(
		ctx,
		m.GitProvider,
		m.Registry,
		canvas.OrganizationID,
		canvasID,
		branch,
		syncDraftBranchOptions{HeadSHA: headSHA, CreatedBy: pushedBy},
	)
	return syncErr
}

// ReconcileBranchDeletion removes the database projection of a draft branch that
// has been deleted from git. It is the worker entry point for "deleted" branch
// notifications: the handler deletes the branch in git and publishes the event,
// then the worker drops the corresponding workflow_versions row and its staging.
// Reconciliation is keyed off git (the source of truth), so it is idempotent and
// only removes rows whose branch no longer exists. The triggering notification is
// already fanned out to clients by the event distributer, so nothing is
// republished here.
func (m *BranchMaterializer) ReconcileBranchDeletion(ctx context.Context, canvasID uuid.UUID, branch string) error {
	_, err := reconcileDraftBranchDeletionsFromGit(ctx, m.GitProvider, canvasID, reconcileDraftBranchDeletionsOptions{BranchName: branch})
	return err
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
