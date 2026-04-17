package roles

import (
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type listCommand struct{}

func (c *listCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.RolesAPI.
		RolesListRoles(ctx.Context).
		DomainType(string(organizationDomainType())).
		DomainId(organizationID).
		Execute()
	if err != nil {
		return err
	}

	roles := response.GetRoles()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(roles)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderRoleListText(stdout, roles)
	})
}
