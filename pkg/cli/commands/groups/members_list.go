package groups

import (
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type membersListCommand struct{}

func (c *membersListCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.GroupsAPI.
		GroupsListGroupUsers(ctx.Context, ctx.Args[0]).
		DomainType(string(organizationDomainType())).
		DomainId(organizationID).
		Execute()
	if err != nil {
		return err
	}

	users := response.GetUsers()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(users)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderGroupUsersText(stdout, users)
	})
}
