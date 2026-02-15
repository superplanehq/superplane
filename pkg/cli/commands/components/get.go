package components

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type getCommand struct{}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	name := ctx.Args[0]
	var component openapi_client.ComponentsComponent

	integrationName, componentName, scoped := core.ParseIntegrationScopedName(name)
	if scoped {
		integration, err := core.FindIntegrationDefinition(ctx, integrationName)
		if err != nil {
			return err
		}

		resolvedComponent, err := findIntegrationComponent(integration, componentName)
		if err != nil {
			return err
		}
		component = resolvedComponent
	} else {
		response, _, err := ctx.API.ComponentAPI.ComponentsDescribeComponent(ctx.Context, name).Execute()
		if err != nil {
			return err
		}
		component = response.GetComponent()
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, _ = fmt.Fprintf(stdout, "Name: %s\n", component.GetName())
			_, _ = fmt.Fprintf(stdout, "Label: %s\n", component.GetLabel())
			_, err := fmt.Fprintf(stdout, "Description: %s\n", component.GetDescription())
			return err
		})
	}

	return ctx.Renderer.Render(component)
}

func findIntegrationComponent(integration openapi_client.IntegrationsIntegrationDefinition, name string) (openapi_client.ComponentsComponent, error) {
	for _, component := range integration.GetComponents() {
		componentName := component.GetName()
		if componentName == name || componentName == fmt.Sprintf("%s.%s", integration.GetName(), name) {
			return component, nil
		}
	}

	return openapi_client.ComponentsComponent{}, fmt.Errorf("component %q not found in integration %q", name, integration.GetName())
}
