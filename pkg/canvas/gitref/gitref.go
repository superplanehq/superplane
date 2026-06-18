// Package gitref holds the canonical canvas git repository vocabulary: the spec
// file names and the draft branch naming/lookup helpers shared by the
// materialization engine and the git-repo seeding/backfill code. It is a leaf
// package (no database or engine dependencies) so both layers can import it
// without creating a cycle.
package gitref

import (
	"context"
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

// NewDraftBranchName returns a fresh, unique draft branch name. The suffix is a
// random UUID, so draft branches do not encode any ownership and never collide;
// the owner is tracked on the materialized version row instead.
func NewDraftBranchName() string {
	return DraftBranchPrefix + uuid.NewString()
}

// GitBranchExists reports whether a branch ref exists in the git repository.
func GitBranchExists(ctx context.Context, gitProvider git.Provider, repoID, branch string) bool {
	_, err := gitProvider.Head(ctx, repoID, branch)
	return err == nil
}
