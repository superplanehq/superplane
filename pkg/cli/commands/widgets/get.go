package widgets

import (
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type getCommand struct {
	canvasID *string
}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, valueOf(c.canvasID))
	if err != nil {
		return err
	}

	canvas, err := fetchCanvas(ctx, canvasID)
	if err != nil {
		return err
	}

	node, err := findWidgetNode(canvas, ctx.Args[0])
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(node)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderNodeText(stdout, node)
	})
}
