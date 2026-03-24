package runs

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type CancelRunCommand struct {
	CanvasID *string
	RunID    *string
}

func (c *CancelRunCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, *c.CanvasID)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasNodeExecutionAPI.
		CanvasesCancelExecution(ctx.Context, canvasID, *c.RunID).
		Body(map[string]any{}).
		Execute()

	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Run cancelled: %s\n", *c.RunID)
		return err
	})
}
