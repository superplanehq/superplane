package index

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "index",
		Short: "Discover available integrations, triggers, components, and widgets",
	}

	root.AddCommand(newIntegrationsCommand(options))
	root.AddCommand(newTriggersCommand(options))
	root.AddCommand(newComponentsCommand(options))
	root.AddCommand(newWidgetsCommand(options))

	return root
}
