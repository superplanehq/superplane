package canvases

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type deleteCommand struct{}

func (c *deleteCommand) Execute(ctx core.CommandContext) error {
	nameOrID := ctx.Args[0]

	canvasID, err := findCanvasID(ctx, ctx.API, nameOrID)
	if err != nil {
		return err
	}

	_, _, err = ctx.API.CanvasAPI.
		CanvasesDeleteCanvas(ctx.Context, canvasID).
		Execute()
	if err != nil {
		return err
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, err := fmt.Fprintf(stdout, "Canvas deleted: %s\n", nameOrID)
			return err
		})
	}

	return ctx.Renderer.Render(map[string]string{
		"id":      canvasID,
		"deleted": "true",
	})
}
