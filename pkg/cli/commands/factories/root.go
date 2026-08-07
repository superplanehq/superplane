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

	artifactCmd := &cobra.Command{
		Use:     "artifact",
		Short:   "Manage work order artifacts",
		Aliases: []string{"artifacts"},
	}

	var (
		title  string
		body   string
		file   string
		url    string
		number int64
		name   string
	)

	addCmd := &cobra.Command{
		Use:   "add <factory> <order-id> <type>",
		Short: "Attach an artifact to a work order",
		Long: `Attach a typed artifact to a work order.

<factory> is a factory name or UUID. <order-id> is the work order UUID.
<type> is one of: pr, markdown, branch.

Examples:
  superplane factory artifact add shipping "$OID" markdown --title PLAN.md -f ./PLAN.md
  superplane factory artifact add shipping "$OID" pr --url https://github.com/org/repo/pull/7 --number 7
  superplane factory artifact add shipping "$OID" branch --name feature/login`,
		Args: cobra.ExactArgs(3),
	}
	addCmd.Flags().StringVar(&title, "title", "", "artifact title")
	addCmd.Flags().StringVar(&body, "body", "", "markdown body (inline)")
	addCmd.Flags().StringVarP(&file, "file", "f", "", "read markdown body from file (or - for stdin)")
	addCmd.Flags().StringVar(&url, "url", "", "artifact URL (required for pr)")
	addCmd.Flags().Int64Var(&number, "number", 0, "pull request number")
	addCmd.Flags().StringVar(&name, "name", "", "branch name (required for branch)")
	core.Bind(addCmd, &artifactAddCommand{
		title:  &title,
		body:   &body,
		file:   &file,
		url:    &url,
		number: &number,
		name:   &name,
	}, options)

	listCmd := &cobra.Command{
		Use:   "list <factory> <order-id>",
		Short: "List artifacts on a work order",
		Args:  cobra.ExactArgs(2),
	}
	core.Bind(listCmd, &artifactListCommand{}, options)

	artifactCmd.AddCommand(addCmd)
	artifactCmd.AddCommand(listCmd)
	root.AddCommand(artifactCmd)

	return root
}
