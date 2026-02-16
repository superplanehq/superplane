package index

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
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
		_, _ = fmt.Fprintf(stdout, "Name: %s\n", integration.GetName())
		_, _ = fmt.Fprintf(stdout, "Label: %s\n", integration.GetLabel())
		_, _ = fmt.Fprintf(stdout, "Description: %s\n", integration.GetDescription())
		_, _ = fmt.Fprintf(stdout, "Components: %d\n", len(integration.GetComponents()))
		_, err := fmt.Fprintf(stdout, "Triggers: %d\n", len(integration.GetTriggers()))
		return err
	})
}
