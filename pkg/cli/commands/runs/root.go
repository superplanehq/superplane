package runs

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	var appID string
	var limit int64
	var before string
	var states []string
	var results []string

	root := &cobra.Command{
		Use:     "runs",
		Short:   "List and inspect app runs",
		Aliases: []string{"run"},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List runs for an app",
		Args:  cobra.NoArgs,
	}
	core.BindAppIDFlag(listCmd, &appID, "app ID")
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
	core.BindAppIDFlag(describeCmd, &appID, "app ID")
	core.Bind(describeCmd, &DescribeRunCommand{
		AppID: &appID,
	}, options)

	root.AddCommand(listCmd)
	root.AddCommand(describeCmd)

	return root
}
