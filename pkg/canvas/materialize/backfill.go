package materialize

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var backfillCommitAuthor = git.CommitAuthor{
	Name:  "SuperPlane",
	Email: "bot@superplane.local",
}

// BackfillCanvasRepository brings an existing (pre-git-first) canvas repository
// up to the git-first model: it ensures the main branch carries canvas.yaml /
// console.yaml seeded from the live version row, then turns every existing draft
// version row into a real git branch carrying the same spec files.
//
// It is idempotent: branches and spec files that already exist are left
// untouched, so it is safe to run repeatedly during a rollout.
func BackfillCanvasRepository(
	ctx context.Context,
	tx *gorm.DB,
	gitProvider git.Provider,
	orgID uuid.UUID,
	canvasID uuid.UUID,
) error {
	if gitProvider == nil {
		return nil
	}

	repository, err := models.FindRepositoryInTransaction(tx, canvasID)
	if err != nil {
		return fmt.Errorf("repository not found: %w", err)
	}

	canvas, err := models.FindCanvasInTransaction(tx, orgID, canvasID)
	if err != nil {
		return err
	}

	if canvas.LiveVersionID != nil {
		liveVersion, liveErr := models.FindCanvasVersionInTransaction(tx, canvasID, *canvas.LiveVersionID)
		if liveErr != nil {
			return liveErr
		}
		if err := backfillMainBranch(ctx, gitProvider, repository, liveVersion); err != nil {
			return err
		}
	}

	var drafts []models.CanvasVersion
	if err := tx.
		Where("workflow_id = ?", canvasID).
		Where("state = ?", models.CanvasVersionStateDraft).
		Where("branch_name IS NOT NULL AND branch_name <> ''").
		Find(&drafts).Error; err != nil {
		return err
	}

	for i := range drafts {
		if err := BackfillDraftBranch(ctx, tx, gitProvider, orgID, canvasID, &drafts[i]); err != nil {
			return err
		}
	}

	return nil
}

func backfillMainBranch(
	ctx context.Context,
	gitProvider git.Provider,
	repository *models.Repository,
	liveVersion *models.CanvasVersion,
) error {
	if !GitBranchExists(ctx, gitProvider, repository.RepoID, models.CanvasGitBranchMain) {
		_, err := SeedMainRepository(ctx, gitProvider, repository, SeedRepositoryInput{
			Canvas: CanvasYAMLFromVersion(liveVersion),
			Author: backfillCommitAuthor,
		})
		return err
	}

	return seedMissingSpecFiles(ctx, gitProvider, repository.RepoID, models.CanvasGitBranchMain, liveVersion, "Backfill canvas spec files")
}

// BackfillDraftBranch creates a real git branch for an existing DB-native draft
// version and seeds canvas.yaml / console.yaml from the version row when missing.
func BackfillDraftBranch(
	ctx context.Context,
	tx *gorm.DB,
	gitProvider git.Provider,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	version *models.CanvasVersion,
) error {
	if gitProvider == nil || version == nil || version.BranchName == nil {
		return nil
	}

	repository, err := models.FindRepositoryInTransaction(tx, canvasID)
	if err != nil {
		return fmt.Errorf("repository not found: %w", err)
	}

	branchName := *version.BranchName
	if !GitBranchExists(ctx, gitProvider, repository.RepoID, branchName) {
		if err := gitProvider.CreateBranch(ctx, repository.RepoID, branchName, models.CanvasGitBranchMain); err != nil {
			return fmt.Errorf("create draft branch %q: %w", branchName, err)
		}
	}

	return seedMissingSpecFiles(ctx, gitProvider, repository.RepoID, branchName, version, "Backfill draft branch spec files")
}

// seedMissingSpecFiles commits canvas.yaml / console.yaml derived from the given
// version onto the branch, but only for files that are not yet present.
func seedMissingSpecFiles(
	ctx context.Context,
	gitProvider git.Provider,
	repoID string,
	branchName string,
	version *models.CanvasVersion,
	message string,
) error {
	headSHA, err := gitProvider.Head(ctx, repoID, branchName)
	if err != nil {
		return fmt.Errorf("read branch head: %w", err)
	}

	files, err := gitProvider.ListFiles(ctx, repoID, branchName)
	if err != nil {
		return fmt.Errorf("list branch files: %w", err)
	}

	hasCanvas := false
	hasConsole := false
	for _, path := range files {
		switch path {
		case CanvasFileName:
			hasCanvas = true
		case ConsoleFileName:
			hasConsole = true
		}
	}

	if hasCanvas && hasConsole {
		return nil
	}

	ops := make([]git.FileOperation, 0, 2)
	if !hasCanvas {
		canvasYAML, buildErr := BuildCanvasYAMLFromCanvas(CanvasYAMLFromVersion(version))
		if buildErr != nil {
			return buildErr
		}
		ops = append(ops, git.FileOperation{Path: CanvasFileName, Content: bytes.NewReader(canvasYAML), SizeBytes: int64(len(canvasYAML))})
	}
	if !hasConsole {
		consoleYAML, buildErr := BuildConsoleYAMLFromVersion(version)
		if buildErr != nil {
			return buildErr
		}
		ops = append(ops, git.FileOperation{Path: ConsoleFileName, Content: bytes.NewReader(consoleYAML), SizeBytes: int64(len(consoleYAML))})
	}

	if len(ops) == 0 {
		return nil
	}

	_, err = gitProvider.Commit(ctx, repoID, git.CommitOptions{
		Branch:          branchName,
		BaseBranch:      branchName,
		ExpectedHeadSHA: headSHA,
		Message:         message,
		Author:          backfillCommitAuthor,
		Operations:      ops,
	})
	return err
}
