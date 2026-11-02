package tasks

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type artifactRemoveCommand struct {
	workspace *string
	taskID    *string
	key       *string
	title     *string
}

func (c *artifactRemoveCommand) Execute(ctx core.CommandContext) error {
	rawTaskID := strings.TrimSpace(stringValue(c.taskID))
	if rawTaskID == "" {
		return fmt.Errorf("--task is required")
	}

	key := strings.TrimSpace(stringValue(c.key))
	title := strings.TrimSpace(stringValue(c.title))
	if (key == "") == (title == "") {
		return fmt.Errorf("pass --key or --title")
	}

	workspaceID, err := resolveWorkspace(ctx, c.workspace)
	if err != nil {
		return err
	}

	taskID, err := resolveTaskID(rawTaskID)
	if err != nil {
		return err
	}

	body := openapi_client.NewFactoriesRemoveWorkOrderArtifactBody()
	if key != "" {
		body.SetKey(key)
	}
	if title != "" {
		body.SetTitle(title)
	}

	response, _, err := ctx.API.FactoryAPI.
		FactoriesRemoveWorkOrderArtifact(ctx.Context, workspaceID, taskID).
		Body(*body).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		label := strings.TrimSpace(response.GetTitle())
		if label == "" {
			label = strings.TrimSpace(response.GetId())
		}
		_, err := fmt.Fprintf(stdout, "Removed %s.\n", label)
		return err
	})
}
