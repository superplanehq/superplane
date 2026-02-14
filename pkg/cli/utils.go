package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func getOneOrAnotherFlag(cmd *cobra.Command, flag1, flag2 string, required bool) string {
	flag1Value, _ := cmd.Flags().GetString(flag1)
	flag2Value, _ := cmd.Flags().GetString(flag2)

	if flag1Value != "" && flag2Value != "" {
		fmt.Fprintf(os.Stderr, "Error: cannot specify both --%s and --%s\n", flag1, flag2)
		os.Exit(1)
	}

	if flag1Value != "" {
		return flag1Value
	}

	if flag2Value != "" {
		return flag2Value
	}

	if required {
		fmt.Fprintf(os.Stderr, "Error: must specify either --%s or --%s\n", flag1, flag2)
		os.Exit(1)
	}

	return ""
}

func getDomainOrExit(client *openapi_client.APIClient, cmd *cobra.Command) (string, string) {
	response, _, err := client.MeAPI.MeMe(context.Background()).Execute()
	Check(err)

	return string(openapi_client.AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_ORGANIZATION), *response.OrganizationId
}

func findCanvasIDByName(ctx context.Context, client *openapi_client.APIClient, name string) (string, error) {
	response, _, err := client.CanvasAPI.CanvasesListCanvases(ctx).Execute()
	if err != nil {
		return "", err
	}

	var matches []openapi_client.CanvasesCanvas
	for _, canvas := range response.GetCanvases() {
		if canvas.Metadata == nil || canvas.Metadata.Name == nil {
			continue
		}
		if *canvas.Metadata.Name == name {
			matches = append(matches, canvas)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("canvas %q not found", name)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("multiple canvases named %q found", name)
	}

	if matches[0].Metadata == nil || matches[0].Metadata.Id == nil {
		return "", fmt.Errorf("canvas %q is missing an id", name)
	}

	return *matches[0].Metadata.Id, nil
}

func parseIntegrationScopedName(name string) (string, string, bool) {
	integrationName, resourceName, hasDot := strings.Cut(name, ".")
	if !hasDot || integrationName == "" || resourceName == "" {
		return "", "", false
	}

	return integrationName, resourceName, true
}

func listIntegrationDefinitions(ctx context.Context, client *openapi_client.APIClient) ([]openapi_client.IntegrationsIntegrationDefinition, error) {
	response, _, err := client.IntegrationAPI.IntegrationsListIntegrations(ctx).Execute()
	if err != nil {
		return nil, err
	}

	return response.GetIntegrations(), nil
}

func findIntegrationDefinitionByName(
	ctx context.Context,
	client *openapi_client.APIClient,
	name string,
) (openapi_client.IntegrationsIntegrationDefinition, error) {
	integrations, err := listIntegrationDefinitions(ctx, client)
	if err != nil {
		return openapi_client.IntegrationsIntegrationDefinition{}, err
	}

	for _, integration := range integrations {
		if integration.GetName() == name {
			return integration, nil
		}
	}

	return openapi_client.IntegrationsIntegrationDefinition{}, fmt.Errorf("integration %q not found", name)
}

func findIntegrationComponentByName(
	integration openapi_client.IntegrationsIntegrationDefinition,
	name string,
) (openapi_client.ComponentsComponent, error) {
	for _, component := range integration.GetComponents() {
		componentName := component.GetName()
		if componentName == name || componentName == fmt.Sprintf("%s.%s", integration.GetName(), name) {
			return component, nil
		}
	}

	return openapi_client.ComponentsComponent{}, fmt.Errorf("component %q not found in integration %q", name, integration.GetName())
}

func findIntegrationTriggerByName(
	integration openapi_client.IntegrationsIntegrationDefinition,
	name string,
) (openapi_client.TriggersTrigger, error) {
	for _, trigger := range integration.GetTriggers() {
		triggerName := trigger.GetName()
		if triggerName == name || triggerName == fmt.Sprintf("%s.%s", integration.GetName(), name) {
			return trigger, nil
		}
	}

	return openapi_client.TriggersTrigger{}, fmt.Errorf("trigger %q not found in integration %q", name, integration.GetName())
}
