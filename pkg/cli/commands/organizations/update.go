package organizations

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type updateCommand struct {
	name        *string
	description *string
}

func (c *updateCommand) Execute(ctx core.CommandContext) error {
	if !ctx.Cmd.Flags().Changed("name") &&
		!ctx.Cmd.Flags().Changed("description") {
		return fmt.Errorf("at least one flag must be provided: --name or --description")
	}

	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	metadata := openapi_client.OrganizationsOrganizationMetadata{}
	if ctx.Cmd.Flags().Changed("name") {
		metadata.SetName(*c.name)
	}
	if ctx.Cmd.Flags().Changed("description") {
		metadata.SetDescription(*c.description)
	}

	org := openapi_client.OrganizationsOrganization{}
	org.SetMetadata(metadata)

	body := openapi_client.OrganizationsUpdateOrganizationBody{}
	body.SetOrganization(org)

	response, _, err := ctx.API.OrganizationAPI.
		OrganizationsUpdateOrganization(ctx.Context, organizationID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	updated := response.GetOrganization()
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderOrganization(stdout, updated)
	})
}
