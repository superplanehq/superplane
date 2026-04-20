package members

import (
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type inviteLinkGetCommand struct{}

func (c *inviteLinkGetCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.OrganizationAPI.
		OrganizationsGetInviteLink(ctx.Context, organizationID).
		Execute()
	if err != nil {
		return err
	}

	link := response.GetInviteLink()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(link)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderInviteLinkText(stdout, link)
	})
}
