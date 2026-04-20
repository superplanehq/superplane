package groups

import (
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type getCommand struct{}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.GroupsAPI.
		GroupsDescribeGroup(ctx.Context, ctx.Args[0]).
		DomainType(string(core.OrganizationDomainType())).
		DomainId(organizationID).
		Execute()
	if err != nil {
		return err
	}

	group := response.GetGroup()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(group)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderGroupText(stdout, group)
	})
}
