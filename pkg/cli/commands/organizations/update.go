package organizations

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type updateCommand struct {
	name              *string
	description       *string
	versioningEnabled *bool
}

func (c *updateCommand) Execute(ctx core.CommandContext) error {
	if !ctx.Cmd.Flags().Changed("name") &&
		!ctx.Cmd.Flags().Changed("description") &&
		!ctx.Cmd.Flags().Changed("versioning-enabled") {
		return fmt.Errorf("at least one flag must be provided: --name, --description, or --versioning-enabled")
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
	if ctx.Cmd.Flags().Changed("versioning-enabled") {
		metadata.SetVersioningEnabled(*c.versioningEnabled)
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
