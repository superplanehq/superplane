package members

import (
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type inviteLinkUpdateCommand struct {
	enabled *bool
}

func (c *inviteLinkUpdateCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	body := openapi_client.OrganizationsUpdateInviteLinkBody{}
	body.SetEnabled(*c.enabled)

	response, _, err := ctx.API.OrganizationAPI.
		OrganizationsUpdateInviteLink(ctx.Context, organizationID).
		Body(body).
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
