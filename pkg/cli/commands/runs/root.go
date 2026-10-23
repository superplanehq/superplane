package runs

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	var appID string
	var limit int64
	var before string
	var states []string
	var results []string
	var nodeID string
	var executionID string
	var replayTimeout time.Duration
	var replayPollInterval time.Duration

	root := &cobra.Command{
		Use:     "runs",
		Short:   "List and inspect automation runs",
		Aliases: []string{"run"},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List runs for an automation",
		Args:  cobra.NoArgs,
	}
	core.BindAutomationFlag(listCmd, &appID, "automation ID")
	listCmd.Flags().Int64Var(&limit, "limit", 20, "maximum number of items to return")
	listCmd.Flags().StringVar(&before, "before", "", "return items before this timestamp (RFC3339)")
	listCmd.Flags().StringSliceVar(&states, "state", nil, "filter by run state (repeatable, e.g. STATE_STARTED)")
	listCmd.Flags().StringSliceVar(&results, "result", nil, "filter by run result (repeatable, e.g. RESULT_FAILED)")
	core.Bind(listCmd, &ListRunsCommand{
		AppID:   &appID,
		Limit:   &limit,
		Before:  &before,
		States:  &states,
		Results: &results,
	}, options)

	describeCmd := &cobra.Command{
		Use:   "describe [run-id]",
		Short: "Show full details for a run",
		Args:  cobra.ExactArgs(1),
	}
	core.BindAutomationFlag(describeCmd, &appID, "automation ID")
	core.Bind(describeCmd, &DescribeRunCommand{
		AppID: &appID,
	}, options)

	replayCmd := &cobra.Command{
		Use:   "replay",
		Short: "Replay a canvas node from a source execution",
		Args:  cobra.NoArgs,
	}
	core.BindAutomationFlag(replayCmd, &appID, "automation ID")
	replayCmd.Flags().StringVar(&nodeID, "node-id", "", "node ID to replay")
	replayCmd.Flags().StringVar(&executionID, "execution-id", "", "source execution ID to replay from")
	replayCmd.Flags().DurationVar(&replayTimeout, "timeout", 30*time.Second, "how long to wait for the replay's execution to appear before reporting it as queued")
	replayCmd.Flags().DurationVar(&replayPollInterval, "poll-interval", 500*time.Millisecond, "how often to poll for the replay's execution while waiting")
	_ = replayCmd.MarkFlagRequired("node-id")
	_ = replayCmd.MarkFlagRequired("execution-id")
	core.Bind(replayCmd, &ReplayCommand{
		CanvasID:     &appID,
		NodeID:       &nodeID,
		ExecutionID:  &executionID,
		PollTimeout:  &replayTimeout,
		PollInterval: &replayPollInterval,
	}, options)

	root.AddCommand(listCmd)
	root.AddCommand(describeCmd)
	root.AddCommand(replayCmd)

	return root
}
