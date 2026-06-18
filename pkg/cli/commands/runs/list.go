package runs

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type ListRunsCommand struct {
	AppID   *string
	Limit   *int64
	Before  *string
	States  *[]string
	Results *[]string
}

func (c *ListRunsCommand) Execute(ctx core.CommandContext) error {
	appID, err := core.ResolveAppID(ctx, *c.AppID)
	if err != nil {
		return err
	}

	request := ctx.API.CanvasRunAPI.
		CanvasesListRuns(ctx.Context, appID)

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

	if c.States != nil && len(*c.States) > 0 {
		request = request.States(*c.States)
	}

	if c.Results != nil && len(*c.Results) > 0 {
		request = request.Results(*c.Results)
	}

	response, _, err := request.Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		runs := response.GetRuns()
		summary := make([]map[string]any, len(runs))
		for i, run := range runs {
			nodeID, customName := runRootEventFields(run)
			summary[i] = map[string]any{
				"id":         run.GetId(),
				"nodeId":     nodeID,
				"customName": formatRunCustomName(customName),
				"state":      formatRunState(run.GetState()),
				"result":     formatRunResult(run.GetResult()),
				"executions": len(run.GetExecutions()),
				"createdAt":  formatRelativeTime(run.GetCreatedAt()),
			}
		}
		return ctx.Renderer.Render(summary)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		runs := response.GetRuns()
		if len(runs) == 0 {
			_, err := fmt.Fprintln(stdout, "No runs found.")
			return err
		}

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tNODE_ID\tCUSTOM_NAME\tSTATE\tRESULT\tEXECUTIONS\tCREATED")
		for _, run := range runs {
			nodeID, customName := runRootEventFields(run)
			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
				run.GetId(),
				nodeID,
				formatRunCustomName(customName),
				formatRunState(run.GetState()),
				formatRunResult(run.GetResult()),
				len(run.GetExecutions()),
				formatRelativeTime(run.GetCreatedAt()),
			)
		}

		return writer.Flush()
	})
}

func runRootEventFields(run openapi_client.CanvasesCanvasRun) (nodeID, customName string) {
	rootEvent, ok := run.GetRootEventOk()
	if !ok || rootEvent == nil {
		return "", ""
	}

	return rootEvent.GetNodeId(), rootEvent.GetCustomName()
}
