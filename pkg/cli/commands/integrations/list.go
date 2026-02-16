package integrations

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type listCommand struct {
}

func (c *listCommand) Execute(ctx core.CommandContext) error {
	me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return err
	}
	if !me.HasOrganizationId() {
		return fmt.Errorf("organization id not found for authenticated user")
	}

	connectedResponse, _, err := ctx.API.OrganizationAPI.OrganizationsListIntegrations(ctx.Context, me.GetOrganizationId()).Execute()
	if err != nil {
		return err
	}

	availableResponse, _, err := ctx.API.IntegrationAPI.IntegrationsListIntegrations(ctx.Context).Execute()
	if err != nil {
		return err
	}

	integrationsByName := make(map[string]openapi_client.IntegrationsIntegrationDefinition, len(availableResponse.GetIntegrations()))
	for _, integration := range availableResponse.GetIntegrations() {
		integrationsByName[integration.GetName()] = integration
	}

	connected := connectedResponse.GetIntegrations()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(connected)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tNAME\tINTEGRATION\tLABEL\tDESCRIPTION\tSTATE")
		for _, integration := range connected {
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

			_, _ = fmt.Fprintf(
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

		return writer.Flush()
	})
}
