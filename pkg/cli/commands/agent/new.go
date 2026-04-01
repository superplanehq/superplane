package agent

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type NewChatCommand struct {
	CanvasID *string
}

func (c *NewChatCommand) Execute(ctx core.CommandContext) error {
	if !ctx.Renderer.IsText() {
		return fmt.Errorf("agent chat only supports text output")
	}

	canvas, err := c.findCanvas(ctx)
	if err != nil {
		return err
	}

	repl := NewRepl(ReplOptions{
		Canvas: canvas,
	})

	return repl.Run(ctx)
}

func (c *NewChatCommand) findCanvas(ctx core.CommandContext) (*openapi_client.CanvasesCanvas, error) {
	canvasID, err := core.ResolveCanvasID(ctx, valueOrEmpty(c.CanvasID))
	if err != nil {
		return nil, err
	}

	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return nil, err
	}

	return response.Canvas, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
