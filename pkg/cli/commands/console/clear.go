package console

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type clearCommand struct {
	canvasID *string
	yes      *bool
}

func (c *clearCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, valueOf(c.canvasID))
	if err != nil {
		return err
	}

	if !confirmReplace(ctx, c.yes, 0, 0) {
		_, err := fmt.Fprintln(ctx.Cmd.OutOrStdout(), "Aborted.")
		return err
	}

	body := openapi_client.CanvasesUpdateCanvasDashboardBody{}
	body.SetPanels([]openapi_client.CanvasesDashboardPanel{})
	body.SetLayout([]openapi_client.CanvasesDashboardLayoutItem{})

	response, _, err := ctx.API.CanvasAPI.
		CanvasesUpdateCanvasDashboard(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response.GetDashboard())
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Console cleared for canvas %s\n", canvasID)
		return err
	})
}
