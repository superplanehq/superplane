package groups

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

	response, _, err := ctx.API.GroupsAPI.
		GroupsListGroups(ctx.Context).
		DomainType(string(organizationDomainType())).
		DomainId(organizationID).
		Execute()
	if err != nil {
		return err
	}

	items := response.GetGroups()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(items)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderGroupListText(stdout, items)
	})
}
