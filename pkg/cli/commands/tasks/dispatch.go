package tasks

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type taskDispatchCommand struct {
	workspace *string
	taskID    *string
	line      *string
}

func (c *taskDispatchCommand) Execute(ctx core.CommandContext) error {
	taskID := strings.TrimSpace(stringValue(c.taskID))
	if taskID == "" {
		return fmt.Errorf("--task is required")
	}

	lineName := strings.TrimSpace(stringValue(c.line))
	if lineName == "" {
		return fmt.Errorf("--line is required")
	}

	workspaceID, err := resolveWorkspace(ctx, c.workspace)
	if err != nil {
		return err
	}

	body := openapi_client.NewFactoriesDispatchWorkOrderBody()
	body.SetLineName(lineName)

	response, _, err := ctx.API.FactoryAPI.
		FactoriesDispatchWorkOrder(ctx.Context, workspaceID, taskID).
		Body(*body).
		Execute()
	if err != nil {
		return err
	}

	task := response.GetOrder()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(task)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(
			stdout,
			"Task dispatched: %s -> line %q (state: %s)\n",
			task.GetId(),
			lineName,
			formatTaskState(task.GetState()),
		)
		return err
	})
}
