package runs

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type DescribeRunCommand struct {
	AppID *string
}

func (c *DescribeRunCommand) Execute(ctx core.CommandContext) error {
	appID, err := core.ResolveAppID(ctx, *c.AppID)
	if err != nil {
		return err
	}

	runID := ctx.Args[0]
	response, _, err := ctx.API.CanvasRunAPI.
		CanvasesDescribeRun(ctx.Context, appID, runID).
		Execute()
	if err != nil {
		return err
	}

	run, ok := response.GetRunOk()
	if !ok || run == nil {
		return fmt.Errorf("run %q not found", runID)
	}

	rootEvent, ok := run.GetRootEventOk()
	if !ok || rootEvent == nil || rootEvent.GetId() == "" {
		return fmt.Errorf("run %q has no root event", runID)
	}

	executions := run.GetExecutions()

	description := map[string]any{
		"id":         run.GetId(),
		"appId":      run.GetCanvasId(),
		"versionId":  run.GetVersionId(),
		"state":      formatRunState(run.GetState()),
		"result":     formatRunResult(run.GetResult()),
		"createdAt":  formatTimestamp(run.GetCreatedAt()),
		"updatedAt":  formatTimestamp(run.GetUpdatedAt()),
		"finishedAt": formatTimestamp(run.GetFinishedAt()),
		"rootEvent":  rootEvent,
		"executions": describeExecutionRefs(executions),
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(description)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		writeAlignedField(writer, "ID", run.GetId())
		writeAlignedField(writer, "Custom Name", formatRunCustomName(rootEvent.GetCustomName()))
		writeAlignedField(writer, "Node ID", rootEvent.GetNodeId())
		writeAlignedField(writer, "State", formatRunState(run.GetState()))
		writeAlignedField(writer, "Result", formatRunResult(run.GetResult()))
		writeAlignedField(writer, "Created", formatRelativeTime(run.GetCreatedAt()))
		writeAlignedField(writer, "Finished", formatRelativeTime(run.GetFinishedAt()))
		writeAlignedField(writer, "Duration", formatRunDuration(run.GetCreatedAt(), run.GetFinishedAt()))

		if err := writer.Flush(); err != nil {
			return err
		}

		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "Event:")
		writeEventPayload(stdout, rootEvent.GetData())

		_, _ = fmt.Fprintln(stdout)
		if len(executions) == 0 {
			_, err := fmt.Fprintln(stdout, "Executions: none")
			return err
		}

		_, _ = fmt.Fprintln(stdout, "Executions:")
		executionWriter := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(executionWriter, "  ID\tNODE_ID\tSTATE\tRESULT\tCREATED")
		for _, execution := range executions {
			_, _ = fmt.Fprintf(
				executionWriter,
				"  %s\t%s\t%s\t%s\t%s\n",
				execution.GetId(),
				execution.GetNodeId(),
				formatExecutionState(execution.GetState()),
				formatExecutionResult(execution.GetResult()),
				formatRelativeTime(execution.GetCreatedAt()),
			)
		}

		return executionWriter.Flush()
	})
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.Format(time.RFC3339)
}

func describeExecutionRefs(executions []openapi_client.CanvasesCanvasNodeExecutionRef) []map[string]any {
	described := make([]map[string]any, len(executions))
	for i, execution := range executions {
		described[i] = map[string]any{
			"id":        execution.GetId(),
			"nodeId":    execution.GetNodeId(),
			"state":     formatExecutionState(execution.GetState()),
			"result":    formatExecutionResult(execution.GetResult()),
			"createdAt": formatTimestamp(execution.GetCreatedAt()),
		}
	}

	return described
}

func formatRootEventPayload(data map[string]interface{}) string {
	if len(data) == 0 {
		return "-"
	}

	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "-"
	}

	return string(payload)
}

func writeAlignedField(w io.Writer, label, value string) {
	_, _ = fmt.Fprintf(w, "%s\t%s\n", label, value)
}

func writeEventPayload(w io.Writer, data map[string]interface{}) {
	formatted := formatRootEventPayload(data)
	for _, line := range strings.Split(formatted, "\n") {
		_, _ = fmt.Fprintf(w, "  %s\n", line)
	}
}
