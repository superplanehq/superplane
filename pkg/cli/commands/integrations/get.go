package integrations

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type getCommand struct{}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	integration, err := core.FindIntegrationDefinition(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, _ = fmt.Fprintf(stdout, "Name: %s\n", integration.GetName())
			_, _ = fmt.Fprintf(stdout, "Label: %s\n", integration.GetLabel())
			_, _ = fmt.Fprintf(stdout, "Description: %s\n", integration.GetDescription())
			_, _ = fmt.Fprintf(stdout, "Components: %d\n", len(integration.GetComponents()))
			_, err := fmt.Fprintf(stdout, "Triggers: %d\n", len(integration.GetTriggers()))
			return err
		})
	}

	return ctx.Renderer.Render(integration)
}
