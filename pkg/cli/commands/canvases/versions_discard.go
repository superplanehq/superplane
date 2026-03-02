package canvases

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type versionsDiscardCommand struct {
	canvas *string
}

func (c *versionsDiscardCommand) Execute(ctx core.CommandContext) error {
	canvasRef := ""
	if c.canvas != nil {
		canvasRef = *c.canvas
	}

	canvasID, err := resolveCanvasIDFromArgOrActive(ctx, canvasRef)
	if err != nil {
		return err
	}

	versionRef := ""
	if len(ctx.Args) == 1 {
		versionRef = ctx.Args[0]
	}

	versionID, err := resolveWorkingVersionIDFromArgOrActive(ctx, versionRef)
	if err != nil {
		return err
	}

	_, _, err = ctx.API.CanvasVersionAPI.
		CanvasesDiscardCanvasVersion(ctx.Context, canvasID, versionID).
		Execute()
	if err != nil {
		return err
	}

	if ctx.Config != nil && ctx.Config.GetActiveCanvasVersion() == versionID {
		if err := setActiveCanvasAndVersion(ctx, canvasID, ""); err != nil {
			return err
		}
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{
			"canvasId":  canvasID,
			"versionId": versionID,
			"discarded": true,
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Canvas: %s\n", canvasID)
		_, _ = fmt.Fprintf(stdout, "Discarded working version: %s\n", versionID)
		_, err = fmt.Fprintln(stdout, "Done")
		return err
	})
}
