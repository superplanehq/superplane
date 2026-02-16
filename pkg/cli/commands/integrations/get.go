package integrations

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type getCommand struct{}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return err
	}
	if !me.HasOrganizationId() {
		return fmt.Errorf("organization id not found for authenticated user")
	}

	response, _, err := ctx.API.OrganizationAPI.
		OrganizationsDescribeIntegration(ctx.Context, me.GetOrganizationId(), ctx.Args[0]).
		Execute()
	if err != nil {
		return err
	}
	integration := response.GetIntegration()

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(integration)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		metadata := integration.GetMetadata()
		spec := integration.GetSpec()
		status := integration.GetStatus()

		_, _ = fmt.Fprintf(stdout, "ID: %s\n", metadata.GetId())
		_, _ = fmt.Fprintf(stdout, "Name: %s\n", metadata.GetName())
		_, _ = fmt.Fprintf(stdout, "Integration: %s\n", spec.GetIntegrationName())
		_, err := fmt.Fprintf(stdout, "State: %s\n", status.GetState())
		return err
	})
}
