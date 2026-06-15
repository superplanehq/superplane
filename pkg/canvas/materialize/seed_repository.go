package materialize

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
)

type SeedRepositoryInput struct {
	Canvas  *CanvasYAML
	Console *models.ConsoleYAML
	Author  git.CommitAuthor
}

func SeedMainRepository(
	ctx context.Context,
	gitProvider git.Provider,
	repository *models.Repository,
	input SeedRepositoryInput,
) (string, error) {
	if gitProvider == nil {
		return "", fmt.Errorf("git provider is required")
	}
	if repository == nil {
		return "", fmt.Errorf("repository is required")
	}

	if _, err := gitProvider.CreateRepository(ctx, repository.RepoID); err != nil && !strings.Contains(err.Error(), "already") {
		// inmemory provider returns ErrInvalidRepositoryID when repo exists
		if err != git.ErrInvalidRepositoryID {
			return "", err
		}
	}

	if input.Canvas == nil {
		return "", fmt.Errorf("canvas yaml is required")
	}

	canvasYAML, err := BuildCanvasYAMLFromCanvas(input.Canvas)
	if err != nil {
		return "", err
	}

	var consoleYAML []byte
	if input.Console != nil {
		consoleYAML, err = BuildConsoleYAMLFromDashboard(input.Console)
		if err != nil {
			return "", err
		}
	} else {
		consoleYAML, err = BuildEmptyConsoleYAML(repository.CanvasID.String(), input.Canvas.Metadata.Name)
		if err != nil {
			return "", err
		}
	}

	// This is the very first commit on a brand-new repository, so there is no
	// base branch to fork from. Passing BaseBranch here would make the git
	// provider try to branch off a non-existent main and fail.
	return gitProvider.Commit(ctx, repository.RepoID, git.CommitOptions{
		Branch:  models.CanvasGitBranchMain,
		Message: "Initial canvas",
		Author:  input.Author,
		Operations: []git.FileOperation{
			{Path: CanvasFileName, Content: bytes.NewReader(canvasYAML), SizeBytes: int64(len(canvasYAML))},
			{Path: ConsoleFileName, Content: bytes.NewReader(consoleYAML), SizeBytes: int64(len(consoleYAML))},
		},
	})
}

func DefaultDraftBranchName(userID uuid.UUID) string {
	return DraftBranchPrefix + userID.String()
}
