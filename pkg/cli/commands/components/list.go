package components

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type listCommand struct {
	from *string
}

func (c *listCommand) Execute(ctx core.CommandContext) error {
	components := []openapi_client.ComponentsComponent{}

	if c.from != nil && *c.from != "" {
		integration, err := core.FindIntegrationDefinition(ctx, *c.from)
		if err != nil {
			return err
		}
		components = integration.GetComponents()
	} else {
		response, _, err := ctx.API.ComponentAPI.ComponentsListComponents(ctx.Context).Execute()
		if err != nil {
			return err
		}
		components = response.GetComponents()
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
			_, _ = fmt.Fprintln(writer, "NAME\tLABEL\tDESCRIPTION")
			for _, component := range components {
				_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", component.GetName(), component.GetLabel(), component.GetDescription())
			}
			return writer.Flush()
		})
	}

	return ctx.Renderer.Render(components)
}
