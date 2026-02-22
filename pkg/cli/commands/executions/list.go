package executions

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type ListExecutionsCommand struct {
	CanvasID *string
	NodeID   *string
	Limit    *int64
	Before   *string
}

func (c *ListExecutionsCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, *c.CanvasID)
	if err != nil {
		return err
	}

	request := ctx.API.CanvasNodeAPI.
		CanvasesListNodeExecutions(ctx.Context, canvasID, *c.NodeID)

	if c.Limit != nil && *c.Limit > 0 {
		request = request.Limit(*c.Limit)
	}

	if c.Before != nil && *c.Before != "" {
		beforeTime, err := time.Parse(time.RFC3339, *c.Before)
		if err != nil {
			return fmt.Errorf("invalid --before value %q: expected RFC3339 timestamp", *c.Before)
		}
		request = request.Before(beforeTime)
	}

	response, _, err := request.Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tNODE_ID\tSTATE\tRESULT\tMESSAGE\tCREATED_AT\tUPDATED_AT")
		for _, execution := range response.GetExecutions() {
			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				execution.GetId(),
				execution.GetNodeId(),
				execution.GetState(),
				execution.GetResult(),
				stringOrDash(execution.GetResultMessage()),
				execution.GetCreatedAt().Format(time.RFC3339),
				execution.GetUpdatedAt().Format(time.RFC3339),
			)
		}

		return writer.Flush()
	})
}

func stringOrDash(s string) string {
	if s == "" {
		return "-"
	}

	return s
}
