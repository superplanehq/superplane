package tasks

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type artifactListCommand struct {
	workspace *string
	taskID    *string
}

func (c *artifactListCommand) Execute(ctx core.CommandContext) error {
	rawTaskID := strings.TrimSpace(stringValue(c.taskID))

	if rawTaskID == "" {
		return fmt.Errorf("--task is required")
	}

	workspaceID, err := resolveWorkspace(ctx, c.workspace)
	if err != nil {
		return err
	}

	taskID, err := resolveTaskID(rawTaskID)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.FactoryAPI.
		FactoriesListWorkOrderArtifacts(ctx.Context, workspaceID, taskID).
		Execute()
	if err != nil {
		return err
	}

	artifacts := response.GetArtifacts()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(artifacts)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(artifacts) == 0 {
			_, err := fmt.Fprintln(stdout, "No artifacts")
			return err
		}
		for _, artifact := range artifacts {
			title := ""
			if data := artifact.GetData(); data != nil {
				if value, ok := data["title"].(string); ok {
					title = value
				} else if value, ok := data["name"].(string); ok {
					title = value
				} else if value, ok := data["url"].(string); ok {
					title = value
				}
			}
			line := fmt.Sprintf("%s\t%s", artifact.GetId(), artifact.GetType())
			if title != "" {
				line += "\t" + title
			}
			if _, err := fmt.Fprintln(stdout, line); err != nil {
				return err
			}
		}
		return nil
	})
}
