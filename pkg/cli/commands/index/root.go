package index

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "index",
		Short: "Discover available integrations, triggers, components, and widgets",
		Long: `Discover available integrations, triggers, components, and widgets.

Text output is optimized for quick reading and shows top-level fields only. Use -o json
or -o yaml with --name to inspect nested schema fields, enum options, defaults, and
conditions.`,
		Example: `  superplane index actions --name daytona.createRepositorySandbox -o yaml
  superplane index actions --from daytona --full -o json
  superplane index triggers --name github.pull-request -o yaml`,
	}

	root.AddCommand(newIntegrationsCommand(options))
	root.AddCommand(newTriggersCommand(options))
	root.AddCommand(newActionsCommand(options))
	root.AddCommand(newWidgetsCommand(options))
	root.AddCommand(newDumpCommand(options))

	return root
}
