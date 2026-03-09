package canvases

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type publishCommand struct {
	title       *string
	description *string
}

func (c *publishCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := resolveCanvasTarget(ctx)
	if err != nil {
		return err
	}

	versioningContext, err := resolveCanvasVersioningContext(ctx, canvasID)
	if err != nil {
		return err
	}
	if !versioningContext.versioningEnabled {
		return fmt.Errorf("effective canvas versioning is disabled for this canvas")
	}

	draftVersionID, err := findCurrentUserDraftVersionID(ctx, canvasID)
	if err != nil {
		return err
	}
	if draftVersionID == "" {
		return fmt.Errorf("no draft version found; run `superplane canvases update %s --draft ...` first", canvasID)
	}

	body := openapi_client.CanvasesCreateCanvasChangeRequestBody{}
	body.SetVersionId(draftVersionID)

	if c.title != nil {
		trimmedTitle := strings.TrimSpace(*c.title)
		if trimmedTitle != "" {
			body.SetTitle(trimmedTitle)
		}
	}
	if c.description != nil {
		trimmedDescription := strings.TrimSpace(*c.description)
		if trimmedDescription != "" {
			body.SetDescription(trimmedDescription)
		}
	}

	response, _, err := ctx.API.CanvasChangeRequestAPI.
		CanvasesCreateCanvasChangeRequest(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	if response.ChangeRequest == nil {
		return nil
	}

	if ctx.Renderer.IsText() {
		changeRequest := response.ChangeRequest
		metadata := changeRequest.Metadata
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			changeRequestID := ""
			status := ""
			versionID := ""
			if metadata != nil {
				changeRequestID = metadata.GetId()
				status = string(metadata.GetStatus())
				versionID = metadata.GetVersionId()
			}

			_, _ = fmt.Fprintf(stdout, "Change request published: %s\n", changeRequestID)
			_, _ = fmt.Fprintf(stdout, "Status: %s\n", status)
			_, err := fmt.Fprintf(stdout, "Version: %s\n", versionID)
			return err
		})
	}

	return ctx.Renderer.Render(response.ChangeRequest)
}

func resolveCanvasTarget(ctx core.CommandContext) (string, error) {
	if len(ctx.Args) > 1 {
		return "", fmt.Errorf("publish accepts at most one positional argument")
	}

	target := ""
	if len(ctx.Args) == 1 {
		target = strings.TrimSpace(ctx.Args[0])
	} else if ctx.Config != nil {
		target = strings.TrimSpace(ctx.Config.GetActiveCanvas())
	}
	if target == "" {
		return "", fmt.Errorf("<name-or-id> is required (or set an active canvas)")
	}

	return findCanvasID(ctx, ctx.API, target)
}
