package executions

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type CancelExecutionCommand struct {
	CanvasID    *string
	ExecutionID *string
}

func (c *CancelExecutionCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, *c.CanvasID)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasNodeExecutionAPI.
		CanvasesCancelExecution(ctx.Context, canvasID, *c.ExecutionID).
		Body(map[string]any{}).
		Execute()

	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Execution cancelled: %s\n", *c.ExecutionID)
		return err
	})
}
