package automations

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "automations",
		Short:   "Manage automations",
		Long:    core.AgentSkillsHelp() + "\n\nAn automation is the process that runs when work reaches a factory stage.",
		Aliases: []string{"automation"},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List automations",
		Args:  cobra.NoArgs,
	}
	core.Bind(listCmd, &listCommand{}, options)

	activeCmd := &cobra.Command{
		Use:   "active [automation-id]",
		Short: "Set the active automation",
		Long:  "Without arguments, prompts for an automation selection. With an automation ID, sets it directly.",
		Args:  cobra.MaximumNArgs(1),
	}
	core.Bind(activeCmd, &ActiveCommand{}, options)

	root.AddCommand(listCmd)
	root.AddCommand(activeCmd)
	root.AddCommand(NewDeleteCommand(options))

	return root
}
