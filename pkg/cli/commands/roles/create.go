package roles

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type createCommand struct {
	file *string
}

func (c *createCommand) Execute(ctx core.CommandContext) error {
	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}
	if filePath == "" {
		return fmt.Errorf("--file is required")
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
		return fmt.Errorf("role metadata.name is required")
	}

	request := openapi_client.RolesCreateRoleRequest{}
	domain := organizationDomainType()
	request.SetDomainType(domain)
	request.SetDomainId(organizationID)
	request.SetRole(resourceToRole(*resource))

	response, _, err := ctx.API.RolesAPI.
		RolesCreateRole(ctx.Context).
		Body(request).
		Execute()
	if err != nil {
		return err
	}

	created := response.GetRole()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(created)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderRoleText(stdout, created)
	})
}
