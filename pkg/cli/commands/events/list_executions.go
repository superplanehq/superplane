package events

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type ListEventExecutionsCommand struct {
	CanvasID *string
	EventID  *string
}

func (c *ListEventExecutionsCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, *c.CanvasID)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasEventAPI.
		CanvasesListEventExecutions(ctx.Context, canvasID, *c.EventID).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tNODE_ID\tSTATE\tRESULT\tCREATED_AT\tUPDATED_AT")
		for _, execution := range response.GetExecutions() {
			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\t%s\t%s\n",
				execution.GetId(),
				execution.GetNodeId(),
				execution.GetState(),
				execution.GetResult(),
				execution.GetCreatedAt().Format(time.RFC3339),
				execution.GetUpdatedAt().Format(time.RFC3339),
			)
		}

		return writer.Flush()
	})
}
