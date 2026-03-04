package canvases

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type versionsCreateCommand struct{}

func (c *versionsCreateCommand) Execute(ctx core.CommandContext) error {
	canvasRef := ""
	if len(ctx.Args) == 1 {
		canvasRef = ctx.Args[0]
	}

	canvasID, err := resolveCanvasIDFromArgOrActive(ctx, canvasRef)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasVersionAPI.
		CanvasesCreateCanvasVersion(ctx.Context, canvasID).
		Execute()
	if err != nil {
		return err
	}

	if response.Version == nil || response.Version.Metadata == nil {
		return fmt.Errorf("failed to create canvas version")
	}

	versionID := response.Version.Metadata.GetId()
	if err := setActiveCanvasAndVersion(ctx, canvasID, versionID); err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response.Version)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Canvas: %s\n", canvasID)
		_, _ = fmt.Fprintf(stdout, "Edit version: %s\n", versionID)
		_, _ = fmt.Fprintf(stdout, "Revision: %d\n", response.Version.Metadata.GetRevision())
		_, err = fmt.Fprintln(stdout, "Active context updated to edit mode")
		return err
	})
}
