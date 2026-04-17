package roles

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type updateCommand struct {
	file *string
}

func (c *updateCommand) Execute(ctx core.CommandContext) error {
	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}
	if filePath == "" {
		return fmt.Errorf("--file is required")
	}
	if len(ctx.Args) > 0 {
		return fmt.Errorf("update does not accept positional arguments")
	}

	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	resource, err := parseRoleFile(filePath)
	if err != nil {
		return err
	}
	if resource.Metadata == nil || resource.Metadata.GetName() == "" {
		return fmt.Errorf("role metadata.name is required for update")
	}

	body := openapi_client.RolesUpdateRoleBody{}
	domain := core.OrganizationDomainType()
	body.SetDomainType(domain)
	body.SetDomainId(organizationID)
	body.SetRole(resourceToRole(*resource))

	response, _, err := ctx.API.RolesAPI.
		RolesUpdateRole(ctx.Context, resource.Metadata.GetName()).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	updated := response.GetRole()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(updated)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderRoleText(stdout, updated)
	})
}
