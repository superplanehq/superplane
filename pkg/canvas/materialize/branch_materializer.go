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
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

type BranchMaterializer struct {
	GitProvider    git.Provider
	Registry       *registry.Registry
	Encryptor      crypto.Encryptor
	AuthService    authorization.Authorization
	WebhookBaseURL string
}

// MaterializeBranch materializes the branch tip (live for main, draft otherwise).
// Idempotency: the underlying sync functions skip work when the branch is already
// materialized at headSHA. Primary entry point for the worker.
func (m *BranchMaterializer) MaterializeBranch(ctx context.Context, canvasID uuid.UUID, branch, headSHA string, pushedBy *uuid.UUID) error {
	headSHA = strings.TrimSpace(headSHA)
	if headSHA == "" {
		return fmt.Errorf("head sha is required to materialize canvas %s branch %s", canvasID, branch)
	}

	canvas, err := models.FindCanvasWithoutOrgScope(canvasID)
	if err != nil {
		return err
	}

	// Best-effort healing sweep for branch deletions as they will not self-heal.
	// A failure here must not block materializing the incoming branch.
	if reconcileErr := m.sweepDeletedDraftBranchesFromGit(ctx, canvasID); reconcileErr != nil {
		log.Errorf("failed to reconcile draft branch deletions for canvas %s: %v", canvasID, reconcileErr)
	}

	if branch == models.CanvasGitBranchMain {
		// Performance optimization: an unlocked pre-check. The authoritative
		// idempotency guard runs under the branch lock inside syncLiveFromGit.
		if m.liveAlreadyMaterialized(canvasID, strings.TrimSpace(headSHA)) {
			return nil
		}

		_, syncErr := m.syncLiveFromGit(
			ctx,
			canvas.OrganizationID,
			canvasID,
			strings.TrimSpace(headSHA),
		)
		return syncErr
	}

	// Performance optimization: same as m.liveAlreadyMaterialized(...)
	if m.draftAlreadyMaterialized(canvasID, branch, headSHA) {
		return nil
	}

	_, syncErr := m.syncDraftBranchFromGit(
		ctx,
		canvas.OrganizationID,
		canvasID,
		branch,
		headSHA,
		pushedBy,
		syncDraftBranchOptions{},
	)
	return syncErr
}

// ReconcileBranchDeletion drops the database projection of a single draft branch
// that has been deleted from git and notifies clients about the removed version.
// A secondary entry point for the worker.
func (m *BranchMaterializer) ReconcileBranchDeletion(ctx context.Context, canvasID uuid.UUID, branch string) error {
	removed, err := deleteDraftBranch(canvasID, branch)
	if err != nil {
		return err
	}

	if removed != uuid.Nil {
		publishCanvasVersionUpdated(canvasID.String(), removed.String())
	}

	return nil
}

// syncLiveFromGit materializes the main branch tip from git into the live DB.
// Safe to call repeatedly; ignored when git main does not point at headSHA.
func (m *BranchMaterializer) syncLiveFromGit(
	ctx context.Context,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	headSHA string,
) (*models.CanvasVersion, error) {
	repository, skip, err := m.resolveBranchHead(ctx, canvasID, models.CanvasGitBranchMain, headSHA)
	if err != nil || skip {
		return nil, err
	}

	snapshot, loadErr := loadRepoSnapshot(ctx, m.GitProvider, m.Registry, orgID, repository.RepoID, headSHA)
	if loadErr != nil {
		versionID := persistLiveMaterializationError(canvasID, headSHA, loadErr)
		if versionID != uuid.Nil {
			publishCanvasVersionUpdated(canvasID.String(), versionID.String())
		}
		return nil, loadErr
	}

	live := &liveMaterializer{
		GitProvider:    m.GitProvider,
		Registry:       m.Registry,
		Encryptor:      m.Encryptor,
		AuthService:    m.AuthService,
		WebhookBaseURL: m.WebhookBaseURL,
	}

	version, txErr := live.persist(ctx, orgID, canvasID, headSHA, snapshot)
	if txErr != nil {
		versionID := persistLiveMaterializationError(canvasID, headSHA, txErr)
		if versionID != uuid.Nil {
			publishCanvasVersionUpdated(canvasID.String(), versionID.String())
		}
		return nil, txErr
	}

	return version, nil
}

type syncDraftBranchOptions struct {
	DisplayNameOverride string
}

// syncDraftBranchFromGit and materializes the branch tip from git.
// Safe to call repeatedly; ignored when the branch does not point at headSHA.
func (m *BranchMaterializer) syncDraftBranchFromGit(
	ctx context.Context,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	branchName string,
	headSHA string,
	createdBy *uuid.UUID,
	opts syncDraftBranchOptions,
) (*models.CanvasVersion, error) {
	if !models.IsDraftBranch(branchName) {
		return nil, fmt.Errorf("branch %q is not a draft branch", branchName)
	}

	repository, skip, err := m.resolveBranchHead(ctx, canvasID, branchName, headSHA)
	if err != nil || skip {
		return nil, err
	}

	snapshot, loadErr := loadRepoSnapshot(ctx, m.GitProvider, m.Registry, orgID, repository.RepoID, headSHA)
	if loadErr != nil {
		versionID := persistDraftMaterializationError(canvasID, branchName, createdBy, headSHA, loadErr)
		if versionID != uuid.Nil {
			publishCanvasVersionUpdated(canvasID.String(), versionID.String())
		}
		return nil, loadErr
	}

	draft := &draftMaterializer{GitProvider: m.GitProvider, Registry: m.Registry}

	return draft.persist(canvasID, branchName, headSHA, createdBy, opts.DisplayNameOverride, snapshot)
}

// sweepDeletedDraftBranchesFromGit removes every draft workflow_versions row whose
// git ref no longer exists in the repository and notifies clients so the dropped
// versions are refetched. Safe to call repeatedly.
func (m *BranchMaterializer) sweepDeletedDraftBranchesFromGit(
	ctx context.Context,
	canvasID uuid.UUID,
) error {
	if m.GitProvider == nil {
		return fmt.Errorf("git provider is not configured")
	}

	repository, err := models.FindRepositoryUnscoped(canvasID)
	if err != nil {
		return fmt.Errorf("repository not found: %w", err)
	}

	gitBranches, err := m.GitProvider.ListBranches(ctx, repository.RepoID, models.DraftBranchPrefix)
	if err != nil {
		return fmt.Errorf("failed to list draft branches in git: %w", err)
	}

	gitBranchSet := make(map[string]struct{}, len(gitBranches))
	for _, branch := range gitBranches {
		gitBranchSet[branch] = struct{}{}
	}

	removed, err := sweepDeletedDraftBranches(canvasID, gitBranchSet)
	if err != nil {
		return err
	}

	for _, versionID := range removed {
		publishCanvasVersionUpdated(canvasID.String(), versionID.String())
	}

	return nil
}

// resolveBranchHead validates that headSHA is the current tip of branch in git
// before any work is done. It returns the repository plus a skip flag that is
// true when we're in a no-op situation.
//
// Callers should return early without error when skip is set.
func (m *BranchMaterializer) resolveBranchHead(
	ctx context.Context,
	canvasID uuid.UUID,
	branch string,
	headSHA string,
) (*models.Repository, bool, error) {
	if m.GitProvider == nil {
		return nil, false, fmt.Errorf("git provider is not configured")
	}

	if headSHA == "" {
		return nil, false, fmt.Errorf("head sha is required to materialize canvas %s branch %s", canvasID, branch)
	}

	repository, err := models.FindRepositoryUnscoped(canvasID)
	if err != nil {
		return nil, false, fmt.Errorf("repository not found: %w", err)
	}

	currentHead, err := m.GitProvider.Head(ctx, repository.RepoID, branch)
	if err != nil {
		if errors.Is(err, git.ErrInvalidRef) {
			// The branch was deleted between the notification and now.
			// Nothing to materialize; skip as a no-op.
			log.Infof(
				"skipping materialization for canvas %s: branch %s no longer exists",
				canvasID, branch,
			)
			return repository, true, nil
		}
		return nil, false, fmt.Errorf("failed to read head for branch %s: %w", branch, err)
	}

	if headSHA != currentHead {
		// Stale notification: a newer commit already superseded this SHA on the
		// branch, so projecting it would publish outdated content. Skip as a no-op.
		log.Infof(
			"skipping stale materialization for canvas %s branch %s: notification sha %s is not branch head %s",
			canvasID, branch, headSHA, currentHead,
		)
		return repository, true, nil
	}

	return repository, false, nil
}

func (m *BranchMaterializer) liveAlreadyMaterialized(canvasID uuid.UUID, headSHA string) bool {
	liveVersion, err := models.FindLiveCanvasVersion(canvasID)
	if err != nil {
		return false
	}

	return materializedAt(liveVersion, headSHA)
}

func (m *BranchMaterializer) draftAlreadyMaterialized(canvasID uuid.UUID, branch, headSHA string) bool {
	draftVersion, err := models.FindDraftVersionByBranch(canvasID, branch)
	return err == nil && materializedAt(draftVersion, headSHA)
}
