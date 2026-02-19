package queue

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type DeleteQueueItemCommand struct {
	CanvasID *string
	NodeID   *string
	ItemID   *string
}

func (c *DeleteQueueItemCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, *c.CanvasID)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasNodeAPI.
		CanvasesDeleteNodeQueueItem(ctx.Context, canvasID, *c.NodeID, *c.ItemID).
		Execute()

	if err != nil {
		return err
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, err := fmt.Fprintf(stdout, "Queue item deleted: %s\n", *c.ItemID)
			return err
		})
	}

	return ctx.Renderer.Render(response)
}
