package queue

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type ListQueueItemsCommand struct {
	CanvasID *string
	NodeID   *string
}

func (c *ListQueueItemsCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, *c.CanvasID)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasNodeAPI.
		CanvasesListNodeQueueItems(ctx.Context, canvasID, *c.NodeID).
		Execute()

	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tCREATED_AT\tROOT_EVENT_ID\tSOURCE")

		for _, item := range response.GetItems() {
			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\n",
				item.GetId(),
				item.GetCreatedAt().Format(time.RFC3339),
				*item.RootEvent.Id,
				*item.RootEvent.NodeId,
			)
		}

		return writer.Flush()
	})
}
