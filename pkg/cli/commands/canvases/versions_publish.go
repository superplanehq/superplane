package canvases

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type versionsPublishCommand struct {
	canvas              *string
	expectedLiveVersion *string
}

func (c *versionsPublishCommand) Execute(ctx core.CommandContext) error {
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

	body := openapi_client.CanvasesPublishCanvasVersionBody{}
	if c.expectedLiveVersion != nil {
		expected := strings.TrimSpace(*c.expectedLiveVersion)
		if strings.EqualFold(expected, "auto") {
			liveVersionID, liveErr := findLiveCanvasVersionID(ctx, canvasID)
			if liveErr != nil {
				return liveErr
			}
			body.SetExpectedLiveVersionId(liveVersionID)
		} else if expected != "" {
			body.SetExpectedLiveVersionId(expected)
		}
	}

	response, _, err := ctx.API.CanvasVersionAPI.
		CanvasesPublishCanvasVersion(ctx.Context, canvasID, versionID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	if err := setActiveCanvasAndVersion(ctx, canvasID, ""); err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Canvas: %s\n", canvasID)
		_, _ = fmt.Fprintf(stdout, "Published version: %s\n", versionID)
		if response.Version != nil && response.Version.Metadata != nil {
			_, _ = fmt.Fprintf(stdout, "Revision: %d\n", response.Version.Metadata.GetRevision())
		}
		_, err = fmt.Fprintln(stdout, "Active context switched to live")
		return err
	})
}
