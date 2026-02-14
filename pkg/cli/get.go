package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/models"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// Root describe command
var getCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show details of SuperPlane resources",
	Long:    `Get detailed information about SuperPlane resources.`,
	Aliases: []string{"desc", "get"},
}

var getCanvasCmd = &cobra.Command{
	Use:   "canvas <name-or-id>",
	Short: "Get a canvas",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		nameOrID := args[0]
		client := DefaultClient()
		ctx := context.Background()

		canvasID, err := findCanvasID(ctx, client, nameOrID)
		Check(err)

		response, _, err := client.CanvasAPI.CanvasesDescribeCanvas(ctx, canvasID).Execute()
		Check(err)

		resource := models.CanvasResourceFromCanvas(*response.Canvas)
		writeJSONOutput(resource)
	},
}

var getTriggerName string

var getTriggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "Get a trigger",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if getTriggerName == "" {
			Fail("--name is required")
		}

		client := DefaultClient()
		ctx := context.Background()
		var trigger openapi_client.TriggersTrigger

		integrationName, triggerName, isScoped := parseIntegrationScopedName(getTriggerName)
		if isScoped {
			integration, err := findIntegrationDefinitionByName(ctx, client, integrationName)
			Check(err)
			trigger, err = findIntegrationTriggerByName(integration, triggerName)
			Check(err)
		} else {
			response, _, err := client.TriggerAPI.TriggersDescribeTrigger(ctx, getTriggerName).Execute()
			Check(err)
			trigger = response.GetTrigger()
		}

		writeJSONOutput(trigger)
	},
}

var getComponentName string

var getComponentCmd = &cobra.Command{
	Use:   "component",
	Short: "Get a component",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if getComponentName == "" {
			Fail("--name is required")
		}

		client := DefaultClient()
		ctx := context.Background()
		var component openapi_client.ComponentsComponent

		integrationName, componentName, isScoped := parseIntegrationScopedName(getComponentName)
		if isScoped {
			integration, err := findIntegrationDefinitionByName(ctx, client, integrationName)
			Check(err)
			component, err = findIntegrationComponentByName(integration, componentName)
			Check(err)
		} else {
			response, _, err := client.ComponentAPI.ComponentsDescribeComponent(ctx, getComponentName).Execute()
			Check(err)
			component = response.GetComponent()
		}

		writeJSONOutput(component)
	},
}

func writeJSONOutput(v any) {
	output, err := json.MarshalIndent(v, "", "  ")
	Check(err)

	fmt.Fprintln(os.Stdout, string(output))
}

func findCanvasID(ctx context.Context, client *openapi_client.APIClient, nameOrID string) (string, error) {
	_, err := uuid.Parse(nameOrID)
	if err == nil {
		return nameOrID, nil
	}

	return findCanvasIDByName(ctx, client, nameOrID)
}

func init() {
	RootCmd.AddCommand(getCmd)
	getCmd.AddCommand(getCanvasCmd)
	getCmd.AddCommand(getTriggerCmd)
	getCmd.AddCommand(getComponentCmd)

	getTriggerCmd.Flags().StringVar(&getTriggerName, "name", "", "trigger name")
	getComponentCmd.Flags().StringVar(&getComponentName, "name", "", "component name")
}
