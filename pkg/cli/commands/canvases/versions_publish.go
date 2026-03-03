package canvases

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type versionsPublishCommand struct {
	canvas *string
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

	response, _, err := ctx.API.CanvasChangeRequestAPI.
		CanvasesCreateCanvasChangeRequest(ctx.Context, canvasID).
		Body(openapi_client.CanvasesCreateCanvasChangeRequestBody{
			VersionId: &versionID,
		}).
		Execute()
	if err != nil {
		return err
	}

	if response.ChangeRequest == nil || response.ChangeRequest.Metadata == nil || response.ChangeRequest.Metadata.Id == nil {
		return fmt.Errorf("failed to create change request")
	}

	changeRequestID := response.ChangeRequest.Metadata.GetId()
	publishResponse, _, err := ctx.API.CanvasChangeRequestAPI.
		CanvasesPublishCanvasChangeRequest(ctx.Context, canvasID, changeRequestID).
		Body(map[string]interface{}{}).
		Execute()
	if err != nil {
		return err
	}

	if err := setActiveCanvasAndVersion(ctx, canvasID, ""); err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(publishResponse)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Canvas: %s\n", canvasID)
		_, _ = fmt.Fprintf(stdout, "Published version: %s\n", versionID)
		if publishResponse.Version != nil && publishResponse.Version.Metadata != nil {
			_, _ = fmt.Fprintf(stdout, "Revision: %d\n", publishResponse.Version.Metadata.GetRevision())
		}
		_, _ = fmt.Fprintf(stdout, "Published change request: %s\n", changeRequestID)
		_, err = fmt.Fprintln(stdout, "Active context switched to live")
		return err
	})
}
