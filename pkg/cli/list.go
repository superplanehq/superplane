package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// Root list command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List SuperPlane resources",
	Long:    `List multiple SuperPlane resources.`,
	Aliases: []string{"ls"},
}

var listCanvasCmd = &cobra.Command{
	Use:     "canvas",
	Short:   "List canvases",
	Aliases: []string{"canvases"},
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := DefaultClient()
		ctx := context.Background()
		response, _, err := client.CanvasAPI.CanvasesListCanvases(ctx).Execute()
		Check(err)

		writer := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(writer, "ID\tNAME\tCREATED_AT")
		for _, canvas := range response.GetCanvases() {
			metadata := canvas.GetMetadata()
			createdAt := ""
			if metadata.HasCreatedAt() {
				createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\n", metadata.GetId(), metadata.GetName(), createdAt)
		}
		_ = writer.Flush()
	},
}

var listIntegrationsCmd = &cobra.Command{
	Use:   "integrations",
	Short: "List integrations",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := DefaultClient()
		ctx := context.Background()
		writer := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

		if !listIntegrationsConnected {
			response, _, err := client.IntegrationAPI.IntegrationsListIntegrations(ctx).Execute()
			Check(err)

			fmt.Fprintln(writer, "NAME\tLABEL\tDESCRIPTION")
			for _, integration := range response.GetIntegrations() {
				fmt.Fprintf(writer, "%s\t%s\t%s\n", integration.GetName(), integration.GetLabel(), integration.GetDescription())
			}
			_ = writer.Flush()
			return
		}

		me, _, err := client.MeAPI.MeMe(ctx).Execute()
		Check(err)
		if !me.HasOrganizationId() {
			Fail("organization id not found for authenticated user")
		}

		connectedResponse, _, err := client.OrganizationAPI.OrganizationsListIntegrations(ctx, me.GetOrganizationId()).Execute()
		Check(err)

		availableResponse, _, err := client.IntegrationAPI.IntegrationsListIntegrations(ctx).Execute()
		Check(err)

		integrationsByName := make(map[string]openapi_client.IntegrationsIntegrationDefinition, len(availableResponse.GetIntegrations()))
		for _, integration := range availableResponse.GetIntegrations() {
			integrationsByName[integration.GetName()] = integration
		}

		fmt.Fprintln(writer, "ID\tNAME\tINTEGRATION\tLABEL\tDESCRIPTION\tSTATE")
		for _, integration := range connectedResponse.GetIntegrations() {
			metadata := integration.GetMetadata()
			spec := integration.GetSpec()
			status := integration.GetStatus()
			integrationName := spec.GetIntegrationName()

			definition, found := integrationsByName[integrationName]
			label := ""
			description := ""
			if found {
				label = definition.GetLabel()
				description = definition.GetDescription()
			}

			fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\t%s\t%s\n",
				metadata.GetId(),
				metadata.GetName(),
				integrationName,
				label,
				description,
				status.GetState(),
			)
		}
		_ = writer.Flush()
	},
}

var listComponentsFrom string
var listIntegrationsConnected bool

var listComponentsCmd = &cobra.Command{
	Use:   "components",
	Short: "List components",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := DefaultClient()
		ctx := context.Background()
		components := []openapi_client.ComponentsComponent{}

		if listComponentsFrom == "" {
			response, _, err := client.ComponentAPI.ComponentsListComponents(ctx).Execute()
			Check(err)
			components = response.GetComponents()
		} else {
			integration, err := findIntegrationDefinitionByName(ctx, client, listComponentsFrom)
			Check(err)
			components = integration.GetComponents()
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(writer, "NAME\tLABEL\tDESCRIPTION")
		for _, component := range components {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", component.GetName(), component.GetLabel(), component.GetDescription())
		}
		_ = writer.Flush()
	},
}

var listTriggersFrom string

var listTriggersCmd = &cobra.Command{
	Use:   "triggers",
	Short: "List triggers",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := DefaultClient()
		ctx := context.Background()
		triggers := []openapi_client.TriggersTrigger{}

		if listTriggersFrom == "" {
			response, _, err := client.TriggerAPI.TriggersListTriggers(ctx).Execute()
			Check(err)
			triggers = response.GetTriggers()
		} else {
			integration, err := findIntegrationDefinitionByName(ctx, client, listTriggersFrom)
			Check(err)
			triggers = integration.GetTriggers()
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(writer, "NAME\tLABEL\tDESCRIPTION")
		for _, trigger := range triggers {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", trigger.GetName(), trigger.GetLabel(), trigger.GetDescription())
		}
		_ = writer.Flush()
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listCanvasCmd)
	listCmd.AddCommand(listIntegrationsCmd)
	listCmd.AddCommand(listComponentsCmd)
	listCmd.AddCommand(listTriggersCmd)

	listIntegrationsCmd.Flags().BoolVar(&listIntegrationsConnected, "connected", false, "list connected integrations for the authenticated organization")
	listComponentsCmd.Flags().StringVar(&listComponentsFrom, "from", "", "integration name")
	listTriggersCmd.Flags().StringVar(&listTriggersFrom, "from", "", "integration name")
}
