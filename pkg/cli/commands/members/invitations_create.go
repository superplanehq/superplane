package members

import (
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type invitationsCreateCommand struct {
	email *string
}

func (c *invitationsCreateCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	body := openapi_client.OrganizationsCreateInvitationBody{}
	body.SetEmail(*c.email)

	response, _, err := ctx.API.OrganizationAPI.
		OrganizationsCreateInvitation(ctx.Context, organizationID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	invitation := response.GetInvitation()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(invitation)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderInvitationText(stdout, invitation)
	})
}
