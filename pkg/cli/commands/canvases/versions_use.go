package canvases

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type versionsUseCommand struct {
	canvas *string
}

func (c *versionsUseCommand) Execute(ctx core.CommandContext) error {
	canvasRef := ""
	if c.canvas != nil {
		canvasRef = *c.canvas
	}

	canvasID, err := resolveCanvasIDFromArgOrActive(ctx, canvasRef)
	if err != nil {
		return err
	}

	versionID, isLive, err := resolveVersionRef(ctx, canvasID, ctx.Args[0])
	if err != nil {
		return err
	}

	version, err := describeCanvasVersion(ctx, canvasID, versionID)
	if err != nil {
		return err
	}

	activeVersion := versionID
	if isLive {
		activeVersion = ""
	}

	if err := setActiveCanvasAndVersion(ctx, canvasID, activeVersion); err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(version)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		metadata := version.GetMetadata()
		_, _ = fmt.Fprintf(stdout, "Canvas: %s\n", canvasID)
		if isLive {
			_, _ = fmt.Fprintf(stdout, "Using live version: %s\n", metadata.GetId())
		} else {
			_, _ = fmt.Fprintf(stdout, "Using edit version: %s\n", metadata.GetId())
		}
		_, _ = fmt.Fprintf(stdout, "Revision: %d\n", metadata.GetRevision())
		_, err = fmt.Fprintln(stdout, "Active context updated")
		return err
	})
}
