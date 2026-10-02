package runners

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

var runnerStates = []string{"pending", "idle", "busy", "terminated"}

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "runners",
		Short: "Manage runners",
	}
	root.AddCommand(newCreateCommand(options))
	root.AddCommand(newListCommand(options))
	root.AddCommand(newDescribeCommand(options))
	root.AddCommand(newDeleteCommand(options))
	return root
}

type createCommand struct {
	FleetID        string
	TaskID         string
	IdempotencyKey string
	Ephemeral      bool
}

func newCreateCommand(options core.BindOptions) *cobra.Command {
	handler := &createCommand{}
	cmd := &cobra.Command{Use: "create", Short: "Create a runner", Args: cobra.NoArgs}
	cmd.Flags().StringVar(&handler.FleetID, "fleet", "", "fleet ID")
	cmd.Flags().StringVar(&handler.TaskID, "task", "", "task ID")
	cmd.Flags().StringVar(
		&handler.IdempotencyKey,
		"idempotency-key",
		"",
		"idempotency key",
	)
	cmd.Flags().BoolVar(&handler.Ephemeral, "ephemeral", false, "terminate after one task")
	_ = cmd.MarkFlagRequired("fleet")
	core.Bind(cmd, handler, options)
	return cmd
}

func (c *createCommand) Execute(ctx core.CommandContext) error {
	request := openapi_client.RunnersCreateRunnerBody{}
	request.SetEphemeral(c.Ephemeral)
	if taskID := strings.TrimSpace(c.TaskID); taskID != "" {
		request.SetTaskId(taskID)
	}
	if key := strings.TrimSpace(c.IdempotencyKey); key != "" {
		request.SetIdempotencyKey(key)
	}

	response, _, err := ctx.API.RunnersAPI.
		RunnersCreateRunner(ctx.Context, c.FleetID).
		Body(request).
		Execute()
	if err != nil {
		return core.FormatCommandError(err)
	}
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}
	if response == nil || response.Runner == nil {
		return fmt.Errorf("server returned an empty runner")
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		if _, err := fmt.Fprintln(
			writer,
			"ID\tREGISTRATION_TOKEN\tEXPIRES_AT\tRUNNER_API_URL",
		); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			response.Runner.GetId(),
			response.GetRegistrationToken(),
			formatTime(response.RegistrationExpiresAt),
			response.GetRunnerApiUrl(),
		); err != nil {
			return err
		}
		return writer.Flush()
	})
}

type listCommand struct {
	FleetID string
	States  []string
	Limit   int32
}

func newListCommand(options core.BindOptions) *cobra.Command {
	handler := &listCommand{}
	cmd := &cobra.Command{Use: "list", Short: "List runners in a fleet", Args: cobra.NoArgs}
	cmd.Flags().StringVar(&handler.FleetID, "fleet", "", "fleet ID")
	cmd.Flags().StringSliceVar(&handler.States, "state", nil, "runner state filter (repeatable)")
	cmd.Flags().Int32Var(&handler.Limit, "limit", 200, "maximum number of runners (1-1000)")
	_ = cmd.MarkFlagRequired("fleet")
	core.Bind(cmd, handler, options)
	return cmd
}

func (c *listCommand) Execute(ctx core.CommandContext) error {
	if c.Limit < 1 || c.Limit > 1000 {
		return fmt.Errorf("limit must be between 1 and 1000")
	}
	for _, state := range c.States {
		if !slices.Contains(runnerStates, state) {
			return fmt.Errorf("invalid runner state %q", state)
		}
	}

	request := ctx.API.RunnersAPI.RunnersListRunners(ctx.Context, c.FleetID).Limit(c.Limit)
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
			"ID\tSTATE\tRUNNER_VERSION\tEPHEMERAL\tCREATED_AT\tLAST_SEEN_AT",
		); err != nil {
			return err
		}
		for _, runner := range response.GetRunners() {
			if _, err := fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%t\t%s\t%s\n",
				runner.GetId(),
				runner.GetState(),
				valueOrDash(runner.GetRunnerVersion()),
				runner.GetEphemeral(),
				formatTime(runner.CreatedAt),
				formatTime(runner.LastSeenAt),
			); err != nil {
				return err
			}
		}
		return writer.Flush()
	})
}

type runnerCommand struct {
	FleetID  string
	RunnerID string
}

func bindRunnerCommand(
	use string,
	short string,
	handler *runnerCommand,
	command core.Command,
	options core.BindOptions,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		PreRun: func(_ *cobra.Command, args []string) {
			handler.RunnerID = args[0]
		},
	}
	cmd.Flags().StringVar(&handler.FleetID, "fleet", "", "fleet ID")
	_ = cmd.MarkFlagRequired("fleet")
	core.Bind(cmd, command, options)
	return cmd
}

type describeCommand struct {
	runnerCommand
}

func newDescribeCommand(options core.BindOptions) *cobra.Command {
	handler := &describeCommand{}
	return bindRunnerCommand(
		"describe <runner-id>",
		"Describe a runner",
		&handler.runnerCommand,
		handler,
		options,
	)
}

func (c *describeCommand) Execute(ctx core.CommandContext) error {
	response, _, err := ctx.API.RunnersAPI.
		RunnersDescribeRunner(ctx.Context, c.FleetID, c.RunnerID).
		Execute()
	if err != nil {
		return core.FormatCommandError(err)
	}
	return renderRunner(ctx, response.Runner)
}

type deleteCommand struct {
	runnerCommand
}

func newDeleteCommand(options core.BindOptions) *cobra.Command {
	handler := &deleteCommand{}
	return bindRunnerCommand(
		"delete <runner-id>",
		"Delete a runner",
		&handler.runnerCommand,
		handler,
		options,
	)
}

func (c *deleteCommand) Execute(ctx core.CommandContext) error {
	response, _, err := ctx.API.RunnersAPI.
		RunnersDeleteRunner(ctx.Context, c.FleetID, c.RunnerID).
		Execute()
	if err != nil {
		return core.FormatCommandError(err)
	}
	return renderRunner(ctx, response.Runner)
}

func renderRunner(ctx core.CommandContext, runner *openapi_client.RunnersRunner) error {
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(runner)
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if runner == nil {
			return fmt.Errorf("server returned an empty runner")
		}
		_, err := fmt.Fprintf(
			stdout,
			"ID: %s\nFleet: %s\nState: %s\nRunner version: %s\nEphemeral: %t\nRegistered: %s\nLast seen: %s\nCreated: %s\nUpdated: %s\nTerminated: %s\nTermination reason: %s\n",
			runner.GetId(),
			runner.GetFleetId(),
			runner.GetState(),
			valueOrDash(runner.GetRunnerVersion()),
			runner.GetEphemeral(),
			formatTime(runner.RegisteredAt),
			formatTime(runner.LastSeenAt),
			formatTime(runner.CreatedAt),
			formatTime(runner.UpdatedAt),
			formatTime(runner.TerminatedAt),
			valueOrDash(strings.TrimSpace(runner.GetTerminationReason())),
		)
		return err
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
