package apps

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/canvas"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/console"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/files"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/memory"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/staging"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "apps",
		Short: "Manage apps",
		Long: core.AgentSkillsHelp() + `

An app is a SuperPlane automation made up of a canvas, console, and files.

App URL pattern: {baseURL}/{organizationId}/apps/{appId}
(e.g. https://app.superplane.com/<organization-id>/apps/<app-id>)`,
		Aliases: []string{"app"},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List apps",
		Args:  cobra.NoArgs,
	}
	core.Bind(listCmd, &listCommand{}, options)

	activeCmd := &cobra.Command{
		Use:   "active [app-id]",
		Short: "Set the active app",
		Long:  "Without arguments, prompts for an app selection. With an app ID, sets it directly.",
		Args:  cobra.MaximumNArgs(1),
	}
	core.Bind(activeCmd, &ActiveCommand{}, options)

	root.AddCommand(listCmd)
	root.AddCommand(activeCmd)
	root.AddCommand(NewCreateCommand(options))
	root.AddCommand(NewDeleteCommand(options))
	root.AddCommand(staging.NewCommand(options))
	root.AddCommand(canvas.NewCommand(options))
	root.AddCommand(console.NewCommand(options))
	root.AddCommand(files.NewRootCommand(options))
	root.AddCommand(memory.NewCommand(options))

	return root
}
