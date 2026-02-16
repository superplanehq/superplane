package index

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func newComponentsCommand(options core.BindOptions) *cobra.Command {
	var from string
	var name string

	cmd := &cobra.Command{
		Use:   "components",
		Short: "List or describe available components",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&from, "from", "", "integration definition name")
	cmd.Flags().StringVar(&name, "name", "", "component name")
	core.Bind(cmd, &componentsCommand{from: &from, name: &name}, options)

	return cmd
}

type componentsCommand struct {
	from *string
	name *string
}

func (c *componentsCommand) Execute(ctx core.CommandContext) error {
	name := strings.TrimSpace(*c.name)
	from := strings.TrimSpace(*c.from)

	if name != "" {
		return c.getComponentByName(ctx, name)
	}

	components, err := c.listComponents(ctx, from)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(components)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "NAME\tLABEL\tDESCRIPTION")
		for _, component := range components {
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", component.GetName(), component.GetLabel(), component.GetDescription())
		}
		return writer.Flush()
	})
}

func (c *componentsCommand) getComponentByName(ctx core.CommandContext, name string) error {
	component, err := c.findComponentByName(ctx, name)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(component)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Name: %s\n", component.GetName())
		_, _ = fmt.Fprintf(stdout, "Label: %s\n", component.GetLabel())
		_, err := fmt.Fprintf(stdout, "Description: %s\n", component.GetDescription())
		return err
	})
}

func (c *componentsCommand) listComponents(ctx core.CommandContext, from string) ([]openapi_client.ComponentsComponent, error) {
	//
	// if --from is used, we grab the components from the integration
	//
	if from != "" {
		integration, err := core.FindIntegrationDefinition(ctx, from)
		if err != nil {
			return nil, err
		}

		return integration.GetComponents(), nil
	}

	//
	// Otherwise, we list core components.
	//
	response, _, err := ctx.API.ComponentAPI.ComponentsListComponents(ctx.Context).Execute()
	if err != nil {
		return nil, err
	}
	return response.GetComponents(), nil
}

func (c *componentsCommand) findComponentByName(ctx core.CommandContext, name string) (openapi_client.ComponentsComponent, error) {
	integrationName, componentName, scoped := core.ParseIntegrationScopedName(name)
	if scoped {
		integration, err := core.FindIntegrationDefinition(ctx, integrationName)
		if err != nil {
			return openapi_client.ComponentsComponent{}, err
		}
		return findIntegrationComponent(integration, componentName)
	}

	response, _, err := ctx.API.ComponentAPI.ComponentsDescribeComponent(ctx.Context, name).Execute()
	if err != nil {
		return openapi_client.ComponentsComponent{}, err
	}

	return response.GetComponent(), nil
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
