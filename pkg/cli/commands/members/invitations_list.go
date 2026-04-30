package members

import (
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type invitationsListCommand struct{}

func (c *invitationsListCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.OrganizationAPI.
		OrganizationsListInvitations(ctx.Context, organizationID).
		Execute()
	if err != nil {
		return err
	}

	invitations := response.GetInvitations()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(invitations)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderInvitationListText(stdout, invitations)
	})
}
