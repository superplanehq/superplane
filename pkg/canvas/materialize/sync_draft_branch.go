package materialize

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
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
	branches, err := models.ListDraftBranchesForCanvas(canvasID)
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

// SyncDraftBranchFromGit registers canvas_draft_branches metadata when missing and
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
) (*models.CanvasDraftBranch, error) {
	if !isDraftBranch(branchName) {
		return nil, fmt.Errorf("branch %q is not a draft branch", branchName)
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

	draftBranch, err := models.FindDraftBranchInTransaction(tx, canvasID, branchName)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if draftBranch == nil {
		label := strings.TrimSpace(opts.DisplayNameOverride)
		if label == "" {
			label, err = NextDraftDisplayName(canvasID)
			if err != nil {
				return nil, err
			}
		}

		ownerID := OwnerFromDraftBranchName(branchName)
		if ownerID == nil {
			ownerID = opts.CreatedBy
		}

		draftBranch = &models.CanvasDraftBranch{
			CanvasID:       canvasID,
			OrganizationID: orgID,
			BranchName:     branchName,
			DisplayName:    label,
			OwnerID:        ownerID,
			CreatedBy:      opts.CreatedBy,
			TipSHA:         headSHA,
		}
		if createErr := models.CreateDraftBranchInTransaction(tx, draftBranch); createErr != nil {
			return nil, createErr
		}
	}

	mat := &DraftMaterializer{GitProvider: gitProvider, Registry: reg}
	ownerID := draftBranch.OwnerID
	if _, matErr := mat.MaterializeDraft(ctx, tx, orgID, canvasID, branchName, headSHA, ownerID); matErr != nil {
		return nil, matErr
	}

	synced, err := models.FindDraftBranchInTransaction(tx, canvasID, branchName)
	if err != nil {
		return nil, err
	}

	return synced, nil
}
