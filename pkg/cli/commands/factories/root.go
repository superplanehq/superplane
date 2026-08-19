package factories

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "factory",
		Short:   "Manage factories and work orders",
		Aliases: []string{"factories"},
	}

	activeCmd := &cobra.Command{
		Use:   "active [factory]",
		Short: "Set or show the active factory",
		Long: `Set the active factory used when --factory is omitted.

Pass a factory name or UUID, or run with no args in an interactive terminal
to pick from a list. Non-interactive with no args prints the active factory id.`,
		Args: cobra.MaximumNArgs(1),
	}
	core.Bind(activeCmd, &activeCommand{}, options)

	artifactCmd := &cobra.Command{
		Use:     "artifacts",
		Short:   "Manage work order artifacts",
		Aliases: []string{"artifact"},
	}

	var (
		factory  string
		orderID  string
		typeName string
		title    string
		body     string
		file     string
		url      string
		number   int64
		name     string
	)

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Attach an artifact to a work order",
		Long: `Attach a typed artifact to a work order.

--factory is a factory name or UUID. When omitted, the active factory
from "superplane factory active" is used. --order-id is the work order UUID.
--type is one of: pr, markdown, branch, link.

Examples:
  superplane factory artifacts list --factory shipping --order-id "$OID"

  # Uses the active factory
  superplane factory artifacts add \
    --order-id "$OID" \
    --type markdown \
    --title "PLAN.md" \
    -f ./PLAN.md

  superplane factory artifacts add \
    --factory shipping \
    --order-id "$OID" \
    --type markdown \
    --title "PLAN.md" \
    -f ./PLAN.md

  superplane factory artifacts add \
    --order-id "$OID" \
    --type pr \
    --url https://github.com/org/repo/pull/7 \
    --number 7

  superplane factory artifacts add \
    --order-id "$OID" \
    --type branch \
    --name feature/login

  superplane factory artifacts add \
    --order-id "$OID" \
    --type link \
    --url https://preview.example.com/pr-42 \
    --title Preview`,
		Args: cobra.NoArgs,
	}
	addCmd.Flags().StringVar(&factory, "factory", "", "factory name or UUID (default: active factory)")
	addCmd.Flags().StringVar(&orderID, "order-id", "", "work order UUID")
	addCmd.Flags().StringVar(&typeName, "type", "", "artifact type: pr, markdown, branch, or link")
	addCmd.Flags().StringVar(&title, "title", "", "artifact title")
	addCmd.Flags().StringVar(&body, "body", "", "markdown body (inline)")
	addCmd.Flags().StringVarP(&file, "file", "f", "", "read markdown body from file (or - for stdin)")
	addCmd.Flags().StringVar(&url, "url", "", "artifact URL (required for pr and link)")
	addCmd.Flags().Int64Var(&number, "number", 0, "pull request number")
	addCmd.Flags().StringVar(&name, "name", "", "branch name (required for branch)")
	_ = addCmd.MarkFlagRequired("order-id")
	_ = addCmd.MarkFlagRequired("type")
	core.Bind(addCmd, &artifactAddCommand{
		factory: &factory,
		orderID: &orderID,
		typ:     &typeName,
		title:   &title,
		body:    &body,
		file:    &file,
		url:     &url,
		number:  &number,
		name:    &name,
	}, options)

	listFactory := ""
	listOrderID := ""
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List artifacts on a work order",
		Long: `List artifacts on a work order.

--factory is a factory name or UUID. When omitted, the active factory
from "superplane factory active" is used.

Example:
  superplane factory artifacts list --factory shipping --order-id "$OID"`,
		Args: cobra.NoArgs,
	}
	listCmd.Flags().StringVar(&listFactory, "factory", "", "factory name or UUID (default: active factory)")
	listCmd.Flags().StringVar(&listOrderID, "order-id", "", "work order UUID")
	_ = listCmd.MarkFlagRequired("order-id")
	core.Bind(listCmd, &artifactListCommand{
		factory: &listFactory,
		orderID: &listOrderID,
	}, options)

	artifactCmd.AddCommand(addCmd)
	artifactCmd.AddCommand(listCmd)
	root.AddCommand(activeCmd)
	root.AddCommand(artifactCmd)
	root.AddCommand(newOrdersCommand(options))

	return root
}

func newOrdersCommand(options core.BindOptions) *cobra.Command {
	ordersCmd := &cobra.Command{
		Use:     "orders",
		Short:   "Create, inspect, and manage work orders",
		Aliases: []string{"order"},
	}

	var (
		orderListFactory    string
		orderListAssignees  []string
		orderListStates     []string
		orderListResults    []string
		orderListUnassigned bool
	)

	orderListCmd := &cobra.Command{
		Use:   "list",
		Short: "List work orders in a factory",
		Long: `List work orders in a factory.

--factory is a factory name or UUID. When omitted, the active factory
from "superplane factory active" is used.

--assignees accepts user UUIDs or emails (comma-separated or repeated).
--state and --result accept the proto enum tokens (e.g. STATE_OPEN,
RESULT_COMPLETED) or short case-insensitive names (open, draft, closed /
completed, rejected, failed).

By default, only open work orders are shown. Pass --state all to see
draft, open, and closed work orders.

Examples:
  superplane factory orders list --factory shipping --state open
  superplane factory orders list --assignees alice@example.com --result failed
  superplane factory orders list --unassigned
  superplane factory orders list --state all`,
		Args: cobra.NoArgs,
	}
	orderListCmd.Flags().StringVar(&orderListFactory, "factory", "", "factory name or UUID (default: active factory)")
	orderListCmd.Flags().StringSliceVar(&orderListAssignees, "assignees", nil, "filter by assignee user UUID or email (repeatable)")
	orderListCmd.Flags().StringSliceVar(&orderListStates, "state", nil, "filter by work order state (repeatable, e.g. open or STATE_OPEN); defaults to open when omitted; pass 'all' to include every state")
	orderListCmd.Flags().StringSliceVar(&orderListResults, "result", nil, "filter by work order result (repeatable, e.g. completed or RESULT_COMPLETED)")
	orderListCmd.Flags().BoolVar(&orderListUnassigned, "unassigned", false, "only show work orders with no assignees")
	core.Bind(orderListCmd, &orderListCommand{
		factory:    &orderListFactory,
		assignees:  &orderListAssignees,
		states:     &orderListStates,
		results:    &orderListResults,
		unassigned: &orderListUnassigned,
	}, options)

	var (
		orderDescribeFactory string
		orderDescribeOrderID string
	)

	orderDescribeCmd := &cobra.Command{
		Use:   "describe",
		Short: "Show a work order's details, comments, and event timeline",
		Long: `Show a work order's title, assignees, description, comments, and
timeline of events.

--factory is a factory name or UUID. When omitted, the active factory
from "superplane factory active" is used. --order is the work order UUID
(--order-id is accepted as an alias).

Example:
  superplane factory orders describe --factory shipping --order "$OID"`,
		Args: cobra.NoArgs,
	}
	orderDescribeCmd.Flags().StringVar(&orderDescribeFactory, "factory", "", "factory name or UUID (default: active factory)")
	bindOrderIDFlag(orderDescribeCmd, &orderDescribeOrderID)
	core.Bind(orderDescribeCmd, &orderDescribeCommand{
		factory: &orderDescribeFactory,
		orderID: &orderDescribeOrderID,
	}, options)

	var (
		orderCreateFactory     string
		orderCreateTitle       string
		orderCreateDescription string
		orderCreateFile        string
		orderCreateAssignees   []string
	)

	orderCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a work order",
		Long: `Create a work order in draft state.

--factory is a factory name or UUID. When omitted, the active factory
from "superplane factory active" is used. --title is required.
--description sets the description inline; -f/--file reads it from a
file (or - for stdin) instead; the two are mutually exclusive.

--assignee accepts a user UUID or email and is repeatable. When omitted,
the work order is assigned to the user running the command.

Examples:
  superplane factory orders create --title "Ship the feature" --description "..."

  superplane factory orders create \
    --title "Ship the feature" \
    -f ./description.md \
    --assignee alice@example.com \
    --assignee bob@example.com`,
		Args: cobra.NoArgs,
	}
	orderCreateCmd.Flags().StringVar(&orderCreateFactory, "factory", "", "factory name or UUID (default: active factory)")
	orderCreateCmd.Flags().StringVar(&orderCreateTitle, "title", "", "work order title (required)")
	orderCreateCmd.Flags().StringVar(&orderCreateDescription, "description", "", "work order description (inline)")
	orderCreateCmd.Flags().StringVarP(&orderCreateFile, "file", "f", "", "read description from file (or - for stdin)")
	orderCreateCmd.Flags().StringArrayVar(&orderCreateAssignees, "assignee", nil, "assignee user UUID or email (repeatable); defaults to the current user when omitted")
	core.Bind(orderCreateCmd, &orderCreateCommand{
		factory:     &orderCreateFactory,
		title:       &orderCreateTitle,
		description: &orderCreateDescription,
		file:        &orderCreateFile,
		assignees:   &orderCreateAssignees,
	}, options)

	var (
		orderDispatchFactory string
		orderDispatchOrderID string
		orderDispatchLine    string
	)

	orderDispatchCmd := &cobra.Command{
		Use:   "dispatch",
		Short: "Dispatch a work order to a factory line",
		Long: `Dispatch a work order to a factory line, starting its execution.

--factory is a factory name or UUID. When omitted, the active factory
from "superplane factory active" is used. --order is the work order UUID
(--order-id is accepted as an alias). --line is the target factory line's
name.

A draft work order moves to the open state on its first dispatch.

Example:
  superplane factory orders dispatch --order "$OID" --line build`,
		Args: cobra.NoArgs,
	}
	orderDispatchCmd.Flags().StringVar(&orderDispatchFactory, "factory", "", "factory name or UUID (default: active factory)")
	bindOrderIDFlag(orderDispatchCmd, &orderDispatchOrderID)
	orderDispatchCmd.Flags().StringVar(&orderDispatchLine, "line", "", "factory line name (required)")
	core.Bind(orderDispatchCmd, &orderDispatchCommand{
		factory: &orderDispatchFactory,
		orderID: &orderDispatchOrderID,
		line:    &orderDispatchLine,
	}, options)

	var (
		orderAssignFactory   string
		orderAssignOrderID   string
		orderAssignAssignees []string
	)

	orderAssignCmd := &cobra.Command{
		Use:   "assign",
		Short: "Set a work order's assignees",
		Long: `Set a work order's assignees.

--factory is a factory name or UUID. When omitted, the active factory
from "superplane factory active" is used. --order is the work order UUID
(--order-id is accepted as an alias).

--assignee accepts a user UUID or email and is repeatable; at least one is
required. This command replaces the entire assignee list with the ones
given — it does not add to the existing list.

Example:
  superplane factory orders assign --order "$OID" --assignee alice@example.com --assignee bob@example.com`,
		Args: cobra.NoArgs,
	}
	orderAssignCmd.Flags().StringVar(&orderAssignFactory, "factory", "", "factory name or UUID (default: active factory)")
	bindOrderIDFlag(orderAssignCmd, &orderAssignOrderID)
	orderAssignCmd.Flags().StringArrayVar(&orderAssignAssignees, "assignee", nil, "assignee user UUID or email (repeatable, required); replaces the full assignee list")
	core.Bind(orderAssignCmd, &orderAssignCommand{
		factory:   &orderAssignFactory,
		orderID:   &orderAssignOrderID,
		assignees: &orderAssignAssignees,
	}, options)

	ordersCmd.AddCommand(orderListCmd)
	ordersCmd.AddCommand(orderDescribeCmd)
	ordersCmd.AddCommand(orderCreateCmd)
	ordersCmd.AddCommand(orderDispatchCmd)
	ordersCmd.AddCommand(orderAssignCmd)

	return ordersCmd
}

// bindOrderIDFlag registers --order and a deprecated --order-id alias
// (mirroring core.BindAppIDFlag's --canvas-id alias) so either spelling
// works, keeping this command consistent with the work order's literal
// spec (--order) and the existing "factory artifacts" commands (--order-id).
func bindOrderIDFlag(cmd *cobra.Command, dest *string) {
	const usage = "work order UUID"
	cmd.Flags().StringVar(dest, "order", "", usage)
	cmd.Flags().StringVar(dest, "order-id", "", usage)
	_ = cmd.Flags().MarkDeprecated("order-id", "use --order")
}
