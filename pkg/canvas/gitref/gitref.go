// Package gitref holds the canonical canvas git repository vocabulary: the spec
// file names and the draft branch naming/lookup helpers shared by the
// materialization engine and the git-repo seeding/backfill code. It is a leaf
// package (no database or engine dependencies) so both layers can import it
// without creating a cycle.
package gitref

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	git "github.com/superplanehq/superplane/pkg/git/provider"
)

const (
	CanvasFileName  = "canvas.yaml"
	ConsoleFileName = "console.yaml"

	DraftBranchPrefix = "drafts/"
)

// IsDraftBranch reports whether a git branch is a canvas draft branch.
func IsDraftBranch(branch string) bool {
	return strings.HasPrefix(branch, DraftBranchPrefix)
}

// DefaultDraftBranchName returns the canonical draft branch name for a user.
func DefaultDraftBranchName(userID uuid.UUID) string {
	return DraftBranchPrefix + userID.String()
}

// OwnerFromDraftBranchName returns the user ID encoded in drafts/{uuid} or
// drafts/{uuid}-{suffix} branch names.
func OwnerFromDraftBranchName(branchName string) *uuid.UUID {
	if !IsDraftBranch(branchName) {
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
