package integrations

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "integrations",
		Short: "Manage integrations",
	}

	var connected bool
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List integrations",
		Args:  cobra.NoArgs,
	}
	listCmd.Flags().BoolVar(&connected, "connected", false, "list connected integrations for the authenticated organization")
	core.Bind(listCmd, &listCommand{connected: &connected}, options)

	getCmd := &cobra.Command{
		Use:   "get <integration-name>",
		Short: "Get integration details",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(getCmd, &getCommand{}, options)

	var integrationID string
	var resourceType string
	var parameters string
	listResourcesCmd := &cobra.Command{
		Use:   "list-resources",
		Short: "List integration resources",
		Args:  cobra.NoArgs,
	}
	listResourcesCmd.Flags().StringVar(&integrationID, "id", "", "connected integration id")
	listResourcesCmd.Flags().StringVar(&resourceType, "type", "", "integration resource type")
	listResourcesCmd.Flags().StringVar(&parameters, "parameters", "", "additional comma-separated query parameters (key=value,key2=value2)")
	_ = listResourcesCmd.MarkFlagRequired("id")
	_ = listResourcesCmd.MarkFlagRequired("type")
	core.Bind(listResourcesCmd, &listResourcesCommand{integrationID: &integrationID, resourceType: &resourceType, parameters: &parameters}, options)

	root.AddCommand(listCmd)
	root.AddCommand(getCmd)
	root.AddCommand(listResourcesCmd)

	return root
}
