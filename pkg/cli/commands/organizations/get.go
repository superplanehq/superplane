package organizations

import (
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type getCommand struct{}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := resolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.OrganizationAPI.
		OrganizationsDescribeOrganization(ctx.Context, organizationID).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	org := response.GetOrganization()
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderOrganization(stdout, org)
	})
}
