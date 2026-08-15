package executions

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	var appID string
	var nodeID string
	var executionID string
	var runID string
	var limit int64
	var before string
	var replayTimeout time.Duration
	var replayPollInterval time.Duration

	root := &cobra.Command{
		Use:     "executions",
		Short:   "Manage app node executions",
		Aliases: []string{"execution"},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List executions for an app node",
		Args:  cobra.NoArgs,
	}
	core.BindAppIDFlag(listCmd, &appID, "app ID")
	listCmd.Flags().StringVar(&nodeID, "node-id", "", "node ID")
	listCmd.Flags().Int64Var(&limit, "limit", 20, "maximum number of items to return")
	listCmd.Flags().StringVar(&before, "before", "", "return items before this timestamp (RFC3339)")
	_ = listCmd.MarkFlagRequired("node-id")
	core.Bind(listCmd, &ListExecutionsCommand{
		CanvasID: &appID,
		NodeID:   &nodeID,
		Limit:    &limit,
		Before:   &before,
	}, options)

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel an execution",
		Args:  cobra.NoArgs,
	}
	core.BindAppIDFlag(cancelCmd, &appID, "app ID")
	cancelCmd.Flags().StringVar(&executionID, "execution-id", "", "execution ID")
	_ = cancelCmd.MarkFlagRequired("execution-id")
	core.Bind(cancelCmd, &CancelExecutionCommand{
		CanvasID:    &appID,
		ExecutionID: &executionID,
	}, options)

	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Fetch runner logs for an execution, run, or node",
		Args:  cobra.NoArgs,
	}
	core.BindAppIDFlag(logsCmd, &appID, "app ID")
	logsCmd.Flags().StringVar(&executionID, "execution-id", "", "execution ID")
	logsCmd.Flags().StringVar(&runID, "run-id", "", "run ID")
	logsCmd.Flags().StringVar(&nodeID, "node-id", "", "node ID")
	logsCmd.Flags().Int64Var(&limit, "limit", 200, "maximum number of log records to return per execution")
	core.Bind(logsCmd, &LogsCommand{
		CanvasID:    &appID,
		ExecutionID: &executionID,
		RunID:       &runID,
		NodeID:      &nodeID,
		Limit:       &limit,
	}, options)

	replayCmd := &cobra.Command{
		Use:   "replay",
		Short: "Replay a canvas node from a source execution",
		Args:  cobra.NoArgs,
	}
	core.BindAppIDFlag(replayCmd, &appID, "app ID")
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
	root.AddCommand(cancelCmd)
	root.AddCommand(logsCmd)
	root.AddCommand(replayCmd)

	return root
}
