package triggers

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
	triggers := []openapi_client.TriggersTrigger{}

	if c.from != nil && *c.from != "" {
		integration, err := core.FindIntegrationDefinition(ctx, *c.from)
		if err != nil {
			return err
		}
		triggers = integration.GetTriggers()
	} else {
		response, _, err := ctx.API.TriggerAPI.TriggersListTriggers(ctx.Context).Execute()
		if err != nil {
			return err
		}
		triggers = response.GetTriggers()
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
			_, _ = fmt.Fprintln(writer, "NAME\tLABEL\tDESCRIPTION")
			for _, trigger := range triggers {
				_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", trigger.GetName(), trigger.GetLabel(), trigger.GetDescription())
			}
			return writer.Flush()
		})
	}

	return ctx.Renderer.Render(triggers)
}
