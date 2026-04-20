package members

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

	response, _, err := ctx.API.UsersAPI.
		UsersListUsers(ctx.Context).
		DomainType(string(core.OrganizationDomainType())).
		DomainId(organizationID).
		IncludeRoles(true).
		Execute()
	if err != nil {
		return err
	}

	users := response.GetUsers()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(users)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderMemberListText(stdout, users)
	})
}
