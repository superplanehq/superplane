package members

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type invitationsRemoveCommand struct{}

func (c *invitationsRemoveCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	invitationID := ctx.Args[0]

	response, _, err := ctx.API.OrganizationAPI.
		OrganizationsRemoveInvitation(ctx.Context, organizationID, invitationID).
		Execute()
	if err != nil {
		return err
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, err := fmt.Fprintf(stdout, "Invitation removed: %s\n", invitationID)
			return err
		})
	}

	return ctx.Renderer.Render(response)
}
