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
--type is one of: pr, markdown, branch.

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
    --name feature/login`,
		Args: cobra.NoArgs,
	}
	addCmd.Flags().StringVar(&factory, "factory", "", "factory name or UUID (default: active factory)")
	addCmd.Flags().StringVar(&orderID, "order-id", "", "work order UUID")
	addCmd.Flags().StringVar(&typeName, "type", "", "artifact type: pr, markdown, or branch")
	addCmd.Flags().StringVar(&title, "title", "", "artifact title")
	addCmd.Flags().StringVar(&body, "body", "", "markdown body (inline)")
	addCmd.Flags().StringVarP(&file, "file", "f", "", "read markdown body from file (or - for stdin)")
	addCmd.Flags().StringVar(&url, "url", "", "artifact URL (required for pr)")
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

	return root
}
