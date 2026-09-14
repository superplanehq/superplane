package cli

import (
	"slices"

	"github.com/spf13/cobra"
	tasks "github.com/superplanehq/superplane/pkg/cli/commands/tasks"
	workspaces "github.com/superplanehq/superplane/pkg/cli/commands/workspaces"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func newDeprecatedFactoryCommand(options core.BindOptions) *cobra.Command {
	factoryCmd := &cobra.Command{
		Use:        "factory",
		Short:      "Deprecated alias for workspace and tasks",
		Long:       "Deprecated. Use workspace for the active workspace, and tasks for task commands.",
		Deprecated: "use workspace and tasks",
		Aliases:    []string{"factories"},
	}

	for _, child := range workspaces.NewCommand(options).Commands() {
		factoryCmd.AddCommand(child)
	}

	ordersCmd := tasks.NewCommand(options)
	ordersCmd.Use = "orders"
	ordersCmd.Short = "Deprecated alias for tasks"
	ordersCmd.Deprecated = "use tasks"
	if !slices.Contains(ordersCmd.Aliases, "order") {
		ordersCmd.Aliases = append(ordersCmd.Aliases, "order")
	}
	factoryCmd.AddCommand(ordersCmd)

	return factoryCmd
}
