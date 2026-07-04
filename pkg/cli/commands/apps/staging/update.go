package staging

import (
	"fmt"
	"io"
	"os"
	"strings"

	canvasmodels "github.com/superplanehq/superplane/pkg/cli/commands/apps/canvas/models"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/console"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type updateCommand struct {
	files *[]string
}

func (c *updateCommand) Execute(ctx core.CommandContext) error {
	localFiles := []string{}
	if c.files != nil {
		localFiles = append(localFiles, *c.files...)
	}
	if len(localFiles) == 0 {
		return fmt.Errorf("at least one --file is required")
	}

	appArg := ""
	if len(ctx.Args) == 1 {
		appArg = strings.TrimSpace(ctx.Args[0])
	}

	appID, err := common.ResolveAppNameOrIDArg(ctx, appArg)
	if err != nil {
		return err
	}

	stagedFiles := make([]common.RepositoryFileStaging, 0, len(localFiles))
	stagedPaths := make([]string, 0, len(localFiles))
	for _, localPath := range localFiles {
		trimmedPath := strings.TrimSpace(localPath)
		if trimmedPath == "" {
			return fmt.Errorf("file path is required")
		}

		content, err := os.ReadFile(trimmedPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", trimmedPath, err)
		}

		repositoryPath := common.RepositoryPathFromLocalFile(trimmedPath)
		switch repositoryPath {
		case common.CanvasYAMLRepositoryPath:
			fileCanvasID, err := canvasIDFromCanvasYAML(content)
			if err != nil {
				return err
			}
			if fileCanvasID != appID {
				return fmt.Errorf(
					"canvas.yaml metadata.id %q does not match app %q",
					fileCanvasID,
					appID,
				)
			}
		case common.ConsoleYAMLRepositoryPath:
			if _, err := console.ParseConsoleYAML(content); err != nil {
				return fmt.Errorf("invalid console yaml in %s: %w", trimmedPath, err)
			}
		}

		stagedFiles = append(stagedFiles, common.RepositoryFileStaging{
			Path:    repositoryPath,
			Content: content,
		})
		stagedPaths = append(stagedPaths, repositoryPath)
	}

	if err := common.StageRepositoryFiles(ctx, appID, stagedFiles); err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{
			"appId":       appID,
			"stagedPaths": stagedPaths,
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		for _, path := range stagedPaths {
			if _, err := fmt.Fprintln(stdout, path); err != nil {
				return err
			}
		}
		return nil
	})
}

func canvasIDFromCanvasYAML(content []byte) (string, error) {
	resource, err := canvasmodels.ParseCanvas(content)
	if err != nil {
		return "", fmt.Errorf("invalid canvas yaml: %w", err)
	}
	if resource.Metadata == nil || strings.TrimSpace(resource.Metadata.GetId()) == "" {
		return "", fmt.Errorf("canvas metadata.id is required in canvas.yaml")
	}
	return strings.TrimSpace(resource.Metadata.GetId()), nil
}
