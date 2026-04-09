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
	Full     *bool
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
		if c.Full != nil && *c.Full {
			return ctx.Renderer.Render(response)
		}

		executions := response.GetExecutions()
		summary := make([]map[string]string, len(executions))
		for i, execution := range executions {
			summary[i] = map[string]string{
				"id":        execution.GetId(),
				"nodeId":    execution.GetNodeId(),
				"state":     string(execution.GetState()),
				"result":    string(execution.GetResult()),
				"createdAt": execution.GetCreatedAt().Format(time.RFC3339),
				"updatedAt": execution.GetUpdatedAt().Format(time.RFC3339),
			}
		}
		return ctx.Renderer.Render(summary)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		executions := response.GetExecutions()
		if len(executions) == 0 {
			_, err := fmt.Fprintln(stdout, "No executions found.")
			return err
		}

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tNODE_ID\tSTATE\tRESULT\tCREATED_AT\tUPDATED_AT")
		for _, execution := range executions {
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
