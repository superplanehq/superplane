package repository

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type CommitCommand struct {
	branch          *string
	expectedHeadSHA *string
	message         *string
	paths           *[]string
}

func (c *CommitCommand) Execute(ctx core.CommandContext) error {
	if c.paths == nil || len(*c.paths) == 0 {
		return fmt.Errorf("at least one --path is required")
	}

	canvasArg := ""
	if len(ctx.Args) == 1 {
		canvasArg = strings.TrimSpace(ctx.Args[0])
	}
	if len(ctx.Args) > 1 {
		return fmt.Errorf("commit accepts at most one positional app argument")
	}

	canvasID, err := common.ResolveAppNameOrIDArg(ctx, canvasArg)
	if err != nil {
		return err
	}

	branch := ""
	if c.branch != nil {
		branch = strings.TrimSpace(*c.branch)
	}
	if branch == "" {
		branch, err = common.CurrentUserDraftBranchName(ctx)
		if err != nil {
			return err
		}
		_, err = common.EnsureCurrentUserDraftBranch(ctx, canvasID)
		if err != nil {
			return err
		}
	}

	expectedHead := ""
	if c.expectedHeadSHA != nil {
		expectedHead = strings.TrimSpace(*c.expectedHeadSHA)
	}
	if expectedHead == "" {
		draftBranch, findErr := common.FindDraftBranch(ctx, canvasID, branch)
		if findErr == nil {
			expectedHead = strings.TrimSpace(draftBranch.GetTipSha())
		}
	}

	commitMessage := "Update repository files"
	if c.message != nil {
		if trimmed := strings.TrimSpace(*c.message); trimmed != "" {
			commitMessage = trimmed
		}
	}

	operations := make([]openapi_client.CanvasesCanvasRepositoryFileOperation, 0, len(*c.paths))
	for _, rawPath := range *c.paths {
		path := normalizeCommitPath(rawPath)
		if path == "" {
			continue
		}
		content, readErr := os.ReadFile(rawPath)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", rawPath, readErr)
		}
		op := openapi_client.NewCanvasesCanvasRepositoryFileOperation()
		op.SetPath(path)
		op.SetContent(base64.StdEncoding.EncodeToString(content))
		operations = append(operations, *op)
	}
	if len(operations) == 0 {
		return fmt.Errorf("no file paths to commit")
	}

	commitSHA, err := common.CommitRepositoryFiles(ctx, canvasID, branch, expectedHead, commitMessage, operations)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{
			"canvasId":  canvasID,
			"branch":    branch,
			"commitSha": commitSHA,
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Committed %d file(s) to %s\n", len(operations), branch)
		_, err := fmt.Fprintf(stdout, "Commit SHA: %s\n", commitSHA)
		return err
	})
}

func normalizeCommitPath(path string) string {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	path = strings.TrimLeft(path, "/")
	if path == "" {
		return ""
	}
	return filepath.ToSlash(path)
}
