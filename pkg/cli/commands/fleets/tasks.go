package fleets

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/cli/fleetapi"
)

var taskStates = []string{"pending", "queued", "reserved", "running", "succeeded", "failed", "canceled", "lost"}

type listTasksCommand struct {
	FleetID string
	States  []string
	Limit   int32
	AfterID string
}

func newListTasksCommand(options core.BindOptions) *cobra.Command {
	handler := &listTasksCommand{}
	cmd := &cobra.Command{Use: "list-tasks", Short: "List tasks in a fleet", Args: cobra.NoArgs}
	cmd.Flags().StringVar(&handler.FleetID, "fleet", "", "fleet ID")
	cmd.Flags().StringSliceVar(&handler.States, "state", nil, "task state filter (repeatable)")
	cmd.Flags().Int32Var(&handler.Limit, "limit", 200, "maximum number of tasks (1-1000)")
	cmd.Flags().StringVar(&handler.AfterID, "after-id", "", "return rows after this ID")
	_ = cmd.MarkFlagRequired("fleet")
	core.Bind(cmd, handler, options)
	return cmd
}

func (c *listTasksCommand) Execute(ctx core.CommandContext) error {
	if c.Limit < 1 || c.Limit > 1000 {
		return fmt.Errorf("limit must be between 1 and 1000")
	}
	for _, state := range c.States {
		if !slices.Contains(taskStates, state) {
			return fmt.Errorf("invalid task state %q", state)
		}
	}

	api, err := fleetapi.New(ctx)
	if err != nil {
		return err
	}
	response, err := api.ListTasks(c.FleetID, c.States, c.Limit, strings.TrimSpace(c.AfterID))
	if err != nil {
		return core.FormatCommandError(err)
	}
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		if _, err := fmt.Fprintln(
			writer,
			"ID\tORGANIZATION_ID\tSTATE\tRUNNER_ID\tQUEUED_AT\tSTARTED_AT\tFINISHED_AT",
		); err != nil {
			return err
		}
		for _, task := range response.GetTasks() {
			if _, err := fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				task.GetId(),
				task.GetOrganizationId(),
				task.GetState(),
				valueOrDash(task.GetRunnerId()),
				formatTime(task.QueuedAt),
				formatTime(task.StartedAt),
				formatTime(task.FinishedAt),
			); err != nil {
				return err
			}
		}
		return writer.Flush()
	})
}

func formatTime(value *time.Time) string {
	if value == nil {
		return "-"
	}
	return value.UTC().Format(time.RFC3339)
}
