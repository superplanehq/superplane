package materialize

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type SyncDraftBranchOptions struct {
	HeadSHA             string
	CreatedBy           *uuid.UUID
	DisplayNameOverride string
}

// SyncDraftBranchFromGit registers draft workflow_versions metadata when missing and
// materializes the branch tip from git. Safe to call repeatedly.
func SyncDraftBranchFromGit(
	ctx context.Context,
	tx *gorm.DB,
	gitProvider git.Provider,
	reg *registry.Registry,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	branchName string,
	opts SyncDraftBranchOptions,
) (*models.CanvasVersion, error) {
	if !isDraftBranch(branchName) {
		return nil, fmt.Errorf("branch %q is not a draft branch", branchName)
	}

	if err := lockBranchMaterialization(tx, canvasID, branchName); err != nil {
		return nil, err
	}

	repository, err := models.FindRepositoryInTransaction(tx, canvasID)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %w", err)
	}

	headSHA := strings.TrimSpace(opts.HeadSHA)
	if headSHA == "" {
		headSHA, err = gitProvider.Head(ctx, repository.RepoID, branchName)
		if err != nil {
			return nil, fmt.Errorf("failed to read branch head: %w", err)
		}
	}

	draftVersion, err := models.FindDraftVersionByBranchInTransaction(tx, canvasID, branchName)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if draftVersion == nil {
		label := strings.TrimSpace(opts.DisplayNameOverride)
		if label == "" {
			label, err = nextDraftDisplayNameInTransaction(tx, canvasID)
			if err != nil {
				return nil, err
			}
		}

		ownerID := OwnerFromDraftBranchName(branchName)
		if ownerID == nil {
			ownerID = opts.CreatedBy
		}

		now := time.Now()
		branch := branchName
		draftVersion = &models.CanvasVersion{
			ID:          uuid.New(),
			WorkflowID:  canvasID,
			OwnerID:     ownerID,
			State:       models.CanvasVersionStateDraft,
			BranchName:  &branch,
			DisplayName: label,
			GitBranch:   branchName,
			CreatedAt:   &now,
			UpdatedAt:   &now,
		}
		if createErr := tx.Create(draftVersion).Error; createErr != nil {
			return nil, createErr
		}
	}

	mat := &DraftMaterializer{GitProvider: gitProvider, Registry: reg}
	ownerID := draftVersion.OwnerID
	version, matErr := mat.MaterializeDraft(ctx, tx, orgID, canvasID, branchName, headSHA, ownerID)
	if matErr != nil {
		return nil, matErr
	}

	if version.DisplayName == "" && draftVersion.DisplayName != "" {
		version.DisplayName = draftVersion.DisplayName
		if saveErr := tx.Save(version).Error; saveErr != nil {
			return nil, saveErr
		}
	}

	return version, nil
}

// RegisterPendingDraftOptions configures RegisterPendingDraftVersion.
type RegisterPendingDraftOptions struct {
	CreatedBy           *uuid.UUID
	DisplayNameOverride string
}

// RegisterPendingDraftVersion records (or updates) the draft workflow_versions
// row for branchName in the "pending" materialization state, without loading the
// git snapshot. Handlers call it after committing to git so the response carries
// a stable version ID and an existing row, then ask the materializer worker to
// fill in nodes/edges/console asynchronously (see RequestBranchMaterialization).
//
// The transaction-scoped advisory lock serializes this against the worker
// materializing the same branch, so the worker reuses this row instead of
// inserting a duplicate that would violate idx_workflow_versions_draft_branch.
func RegisterPendingDraftVersion(
	tx *gorm.DB,
	canvasID uuid.UUID,
	branchName string,
	headSHA string,
	opts RegisterPendingDraftOptions,
) (*models.CanvasVersion, error) {
	if !isDraftBranch(branchName) {
		return nil, fmt.Errorf("branch %q is not a draft branch", branchName)
	}

	if err := lockBranchMaterialization(tx, canvasID, branchName); err != nil {
		return nil, err
	}

	now := time.Now()
	existing, err := models.FindDraftVersionByBranchInTransaction(tx, canvasID, branchName)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if existing != nil {
		existing.CommitSHA = headSHA
		existing.GitBranch = branchName
		existing.MaterializationStatus = models.MaterializationStatusPending
		existing.MaterializationError = ""
		existing.UpdatedAt = &now
		if label := strings.TrimSpace(opts.DisplayNameOverride); label != "" {
			existing.DisplayName = label
		}
		if saveErr := tx.Save(existing).Error; saveErr != nil {
			return nil, saveErr
		}
		return existing, nil
	}

	label := strings.TrimSpace(opts.DisplayNameOverride)
	if label == "" {
		label, err = nextDraftDisplayNameInTransaction(tx, canvasID)
		if err != nil {
			return nil, err
		}
	}

	ownerID := OwnerFromDraftBranchName(branchName)
	if ownerID == nil {
		ownerID = opts.CreatedBy
	}

	branch := branchName
	version := &models.CanvasVersion{
		ID:                    uuid.New(),
		WorkflowID:            canvasID,
		OwnerID:               ownerID,
		State:                 models.CanvasVersionStateDraft,
		BranchName:            &branch,
		GitBranch:             branchName,
		DisplayName:           label,
		CommitSHA:             headSHA,
		MaterializationStatus: models.MaterializationStatusPending,
		CreatedAt:             &now,
		UpdatedAt:             &now,
	}
	if createErr := tx.Create(version).Error; createErr != nil {
		return nil, createErr
	}

	return version, nil
}

var draftDisplayNamePattern = regexp.MustCompile(`^Draft #(\d+)$`)

// OwnerFromDraftBranchName returns the user ID encoded in drafts/{uuid} or
// drafts/{uuid}-{suffix} branch names.
func OwnerFromDraftBranchName(branchName string) *uuid.UUID {
	if !strings.HasPrefix(branchName, DraftBranchPrefix) {
		return nil
	}

	rest := strings.TrimPrefix(branchName, DraftBranchPrefix)
	if id, err := uuid.Parse(rest); err == nil {
		return &id
	}

	if len(rest) > 36 {
		if id, err := uuid.Parse(rest[:36]); err == nil && (len(rest) == 36 || rest[36] == '-') {
			return &id
		}
	}

	return nil
}

// NextDraftDisplayName returns a sequential label such as "Draft #1", "Draft #2".
func NextDraftDisplayName(canvasID uuid.UUID) (string, error) {
	return nextDraftDisplayNameInTransaction(database.Conn(), canvasID)
}

func nextDraftDisplayNameInTransaction(tx *gorm.DB, canvasID uuid.UUID) (string, error) {
	branches, err := models.ListAllDraftBranchVersionsForCanvasInTransaction(tx, canvasID)
	if err != nil {
		return "", err
	}

	highest := 0
	for _, branch := range branches {
		matches := draftDisplayNamePattern.FindStringSubmatch(branch.DisplayName)
		if matches == nil {
			continue
		}
		if n, convErr := strconv.Atoi(matches[1]); convErr == nil && n > highest {
			highest = n
		}
	}

	return fmt.Sprintf("Draft #%d", highest+1), nil
}

// UniqueDraftBranchName returns a drafts/* branch name that does not yet exist in git.
func UniqueDraftBranchName(ctx context.Context, gitProvider git.Provider, repoID string, userID uuid.UUID) (string, error) {
	base := DefaultDraftBranchName(userID)

	existing, err := gitProvider.ListBranches(ctx, repoID, base)
	if err != nil {
		return "", err
	}

	existingSet := make(map[string]struct{}, len(existing))
	for _, branch := range existing {
		existingSet[branch] = struct{}{}
	}

	if _, taken := existingSet[base]; !taken {
		return base, nil
	}

	for attempt := 0; attempt < 50; attempt++ {
		candidate := fmt.Sprintf("%s-%s", base, uuid.NewString()[:8])
		if _, taken := existingSet[candidate]; !taken {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not generate a unique draft branch name after multiple attempts")
}

// GitBranchExists reports whether a branch ref exists in the git repository.
func GitBranchExists(ctx context.Context, gitProvider git.Provider, repoID, branch string) bool {
	_, err := gitProvider.Head(ctx, repoID, branch)
	return err == nil
}
