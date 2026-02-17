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
	response, _, err := ctx.API.CanvasNodeExecutionAPI.
		CanvasesCancelExecution(ctx.Context, *c.CanvasID, *c.ExecutionID).
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
