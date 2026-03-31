package canvases

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type createCommand struct {
	file            *string
	autoLayout      *string
	autoLayoutScope *string
	autoLayoutNodes *[]string
}

func (c *createCommand) Execute(ctx core.CommandContext) error {
	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}
	autoLayoutValue := ""
	if c.autoLayout != nil {
		autoLayoutValue = strings.TrimSpace(*c.autoLayout)
	}
	autoLayoutScopeValue := ""
	if c.autoLayoutScope != nil {
		autoLayoutScopeValue = strings.TrimSpace(*c.autoLayoutScope)
	}
	autoLayoutNodeIDs := []string{}
	if c.autoLayoutNodes != nil {
		autoLayoutNodeIDs = append(autoLayoutNodeIDs, *c.autoLayoutNodes...)
	}

	if filePath != "" {
		if len(ctx.Args) > 0 {
			return fmt.Errorf("cannot use <canvas-name> together with --file")
		}
		return c.createFromFile(ctx, filePath, autoLayoutValue, autoLayoutScopeValue, autoLayoutNodeIDs)
	}

	if len(ctx.Args) != 1 {
		return fmt.Errorf("either --file or <canvas-name> is required")
	}

	name := ctx.Args[0]
	resource := models.Canvas{
		APIVersion: core.APIVersion,
		Kind:       models.CanvasKind,
		Metadata:   &openapi_client.CanvasesCanvasMetadata{Name: &name},
		Spec:       models.EmptyCanvasSpec(),
	}

	request := models.CreateCanvasRequestFromCanvas(resource)
	if autoLayoutFlagsWereSet(ctx) {
		autoLayout, parseErr := parseAutoLayout(autoLayoutValue, autoLayoutScopeValue, autoLayoutNodeIDs)
		if parseErr != nil {
			return parseErr
		}
		if autoLayout != nil {
			request.SetAutoLayout(*autoLayout)
		}
	} else {
		request.SetAutoLayout(buildDefaultAutoLayout())
	}

	resp, httpResp, err := ctx.API.CanvasAPI.CanvasesCreateCanvas(ctx.Context).Body(request).Execute()
	return validateAndPrintCreateResponse(ctx, resp, httpResp, err)
}

func (c *createCommand) createFromFile(
	ctx core.CommandContext,
	path string,
	autoLayoutValue string,
	autoLayoutScopeValue string,
	autoLayoutNodeIDs []string,
) error {
	canvas, fileAutoLayout, err := loadCanvasForCreateFromFile(path)
	if err != nil {
		return err
	}

	request := openapi_client.CanvasesCreateCanvasRequest{}
	request.SetCanvas(canvas)

	if autoLayoutFlagsWereSet(ctx) {
		autoLayout, parseErr := parseAutoLayout(autoLayoutValue, autoLayoutScopeValue, autoLayoutNodeIDs)
		if parseErr != nil {
			return parseErr
		}
		if autoLayout != nil {
			if fileAutoLayout != nil {
				return fmt.Errorf("cannot use auto-layout flags with --file when file already defines autoLayout")
			}
			request.SetAutoLayout(*autoLayout)
		}
	} else {
		if fileAutoLayout != nil {
			request.SetAutoLayout(*fileAutoLayout)
		} else {
			request.SetAutoLayout(buildDefaultAutoLayout())
		}
	}

	resp, httpResp, err := ctx.API.CanvasAPI.CanvasesCreateCanvas(ctx.Context).Body(request).Execute()
	return validateAndPrintCreateResponse(ctx, resp, httpResp, err)
}

func validateAndPrintCreateResponse(
	ctx core.CommandContext,
	resp *openapi_client.CanvasesCreateCanvasResponse,
	httpResp *http.Response,
	err error,
) error {
	if err != nil {
		return err
	}

	if httpResp != nil && (httpResp.StatusCode < 200 || httpResp.StatusCode >= 300) {
		return fmt.Errorf("unexpected response status: %s", httpResp.Status)
	}

	if resp == nil || resp.Canvas == nil || resp.Canvas.Metadata == nil || resp.Canvas.Metadata.GetId() == "" {
		return fmt.Errorf("canvas create returned success but no canvas was returned — the request may not have reached the server (check your context URL scheme)")
	}

	canvas := *resp.Canvas
	resource := models.CanvasResourceFromCanvas(canvas)
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(resource)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Canvas %q created (ID: %s)\n", canvas.Metadata.GetName(), canvas.Metadata.GetId())
		return err
	})
}
