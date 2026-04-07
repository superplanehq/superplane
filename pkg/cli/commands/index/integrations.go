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

func newIntegrationsCommand(options core.BindOptions) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "integrations",
		Short: "List or describe available integration definitions",
		Args:  cobra.NoArgs,
	}
	cmd.Flags().StringVar(&name, "name", "", "integration definition name")
	core.Bind(cmd, &integrationsCommand{name: &name}, options)

	return cmd
}

type integrationsCommand struct {
	name *string
}

func (c *integrationsCommand) Execute(ctx core.CommandContext) error {
	name := strings.TrimSpace(*c.name)
	if name != "" {
		return c.getIntegrationByName(ctx, name)
	}

	response, _, err := ctx.API.IntegrationAPI.IntegrationsListIntegrations(ctx.Context).Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response.GetIntegrations())
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "NAME\tLABEL\tDESCRIPTION")
		for _, integration := range response.GetIntegrations() {
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", integration.GetName(), integration.GetLabel(), integration.GetDescription())
		}
		return writer.Flush()
	})
}

func (c *integrationsCommand) getIntegrationByName(ctx core.CommandContext, name string) error {
	integration, err := core.FindIntegrationDefinition(ctx, name)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(integration)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderIntegrationText(stdout, integration)
	})
}

func renderIntegrationText(stdout io.Writer, integration openapi_client.IntegrationsIntegrationDefinition) error {
	_, err := fmt.Fprintf(stdout, "Name: %s\n", integration.GetName())
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "Label: %s\n", integration.GetLabel())
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "Description: %s\n", integration.GetDescription())
	if err != nil {
		return err
	}

	err = renderConfigurationText(stdout, integration.GetConfiguration())
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout)
	if err != nil {
		return err
	}

	err = renderIntegrationComponentsText(stdout, integration.GetComponents())
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout)
	if err != nil {
		return err
	}

	return renderIntegrationTriggersText(stdout, integration.GetTriggers())
}

func renderIntegrationComponentsText(stdout io.Writer, components []openapi_client.ComponentsComponent) error {
	_, err := fmt.Fprintln(stdout, "Components:")
	if err != nil {
		return err
	}

	if len(components) == 0 {
		_, err = fmt.Fprintln(stdout, "  (none)")
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "  NAME\tLABEL\tDESCRIPTION")
	for _, component := range components {
		_, _ = fmt.Fprintf(writer, "  %s\t%s\t%s\n", component.GetName(), component.GetLabel(), component.GetDescription())
	}
	return writer.Flush()
}

func renderIntegrationTriggersText(stdout io.Writer, triggers []openapi_client.TriggersTrigger) error {
	_, err := fmt.Fprintln(stdout, "Triggers:")
	if err != nil {
		return err
	}

	if len(triggers) == 0 {
		_, err = fmt.Fprintln(stdout, "  (none)")
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "  NAME\tLABEL\tDESCRIPTION")
	for _, trigger := range triggers {
		_, _ = fmt.Fprintf(writer, "  %s\t%s\t%s\n", trigger.GetName(), trigger.GetLabel(), trigger.GetDescription())
	}
	return writer.Flush()
}
