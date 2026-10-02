package tasks

import (
	"fmt"
	"io"
	"slices"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

var taskStates = []string{"queued", "reserved", "running", "succeeded", "failed", "canceled", "lost"}

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "tasks",
		Short: "Inspect runner tasks",
	}
	root.AddCommand(newListCommand(options))
	return root
}

type listCommand struct {
	FleetID string
	States  []string
	Limit   int32
}

func newListCommand(options core.BindOptions) *cobra.Command {
	handler := &listCommand{}
	cmd := &cobra.Command{Use: "list", Short: "List tasks in a fleet", Args: cobra.NoArgs}
	cmd.Flags().StringVar(&handler.FleetID, "fleet", "", "fleet ID")
	cmd.Flags().StringSliceVar(&handler.States, "state", nil, "task state filter (repeatable)")
	cmd.Flags().Int32Var(&handler.Limit, "limit", 200, "maximum number of tasks (1-1000)")
	_ = cmd.MarkFlagRequired("fleet")
	core.Bind(cmd, handler, options)
	return cmd
}

func (c *listCommand) Execute(ctx core.CommandContext) error {
	if c.Limit < 1 || c.Limit > 1000 {
		return fmt.Errorf("limit must be between 1 and 1000")
	}
	for _, state := range c.States {
		if !slices.Contains(taskStates, state) {
			return fmt.Errorf("invalid task state %q", state)
		}
	}

	request := ctx.API.RunnersAPI.RunnersListFleetTasks(ctx.Context, c.FleetID).Limit(c.Limit)
	if len(c.States) > 0 {
		request = request.States(c.States)
	}
	response, _, err := request.Execute()
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

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func formatTime(value *time.Time) string {
	if value == nil {
		return "-"
	}
	return value.UTC().Format(time.RFC3339)
}
