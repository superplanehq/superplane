package organizations

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "organizations",
		Short:   "Manage organizations",
		Aliases: []string{"org", "orgs"},
	}

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get details for the current organization",
		Args:  cobra.NoArgs,
	}
	core.Bind(getCmd, &getCommand{}, options)

	var updateName string
	var updateDescription string
	var updateVersioningEnabled bool
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update the current organization",
		Args:  cobra.NoArgs,
	}
	updateCmd.Flags().StringVar(&updateName, "name", "", "organization name")
	updateCmd.Flags().StringVar(&updateDescription, "description", "", "organization description")
	updateCmd.Flags().BoolVar(&updateVersioningEnabled, "versioning-enabled", false, "enable or disable global versioning")
	core.Bind(updateCmd, &updateCommand{
		name:              &updateName,
		description:       &updateDescription,
		versioningEnabled: &updateVersioningEnabled,
	}, options)

	root.AddCommand(getCmd)
	root.AddCommand(updateCmd)

	return root
}
