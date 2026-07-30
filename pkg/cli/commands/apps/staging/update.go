package staging

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/yaml"
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
			if _, err := yaml.CanvasFromYAML(content); err != nil {
				return fmt.Errorf("invalid canvas yaml in %s: %w", trimmedPath, err)
			}
		case common.ConsoleYAMLRepositoryPath:
			// Lenient parse + shape check: caps are enforced at commit
			// time (see commit_canvas_staging.go, which runs a delta
			// check against the previously committed pages). Staging is
			// a scratchpad, so a grandfathered console with more than
			// MaxConsolePanelsPerPage panels must be re-stageable here
			// even without any authoring changes — otherwise a user who
			// only wants to *reduce* an over-cap page cannot get their
			// edit past this pre-flight. The shape check still surfaces
			// malformed YAML, wrong apiVersion/kind, mixed spec shapes,
			// or unsupported panel types.
			parsed, err := yaml.ConsoleFromYMLLenient(content)
			if err != nil {
				return fmt.Errorf("invalid console yaml in %s: %w", trimmedPath, err)
			}
			if err := parsed.ValidateShape(); err != nil {
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
