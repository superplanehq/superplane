package changes

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/cli/layout"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type ResolveCommand struct {
	file            *string
	autoLayout      *string
	autoLayoutScope *string
	autoLayoutNodes *[]string
}

func (c *ResolveCommand) Execute(ctx core.CommandContext) error {
	changeRequestID, canvasTarget, err := parseCanvasChangeRequestTargetArgs(ctx.Args)
	if err != nil {
		return err
	}

	canvasID, err := resolveCanvasTargetFromOptionalArg(ctx, canvasTarget)
	if err != nil {
		return err
	}

	filePath := ""
	if c.file != nil {
		filePath = strings.TrimSpace(*c.file)
	}
	if filePath == "" {
		return fmt.Errorf("--file is required")
	}

	canvas, err := loadCanvasForChangeRequestResolve(filePath)
	if err != nil {
		return err
	}

	body := openapi_client.CanvasesResolveCanvasChangeRequestBody{}
	body.SetCanvas(canvas)

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

	if layout.HasFlags(ctx) {
		if autoLayoutValue == "" && (autoLayoutScopeValue != "" || len(autoLayoutNodeIDs) > 0) {
			return fmt.Errorf("--auto-layout is required when using --auto-layout-scope or --auto-layout-node")
		}

		if autoLayoutValue != "" {
			autoLayout, parseErr := layout.ParseAutoLayout(autoLayoutValue, autoLayoutScopeValue, autoLayoutNodeIDs)
			if parseErr != nil {
				return parseErr
			}
			body.SetAutoLayout(*autoLayout)
		}
	}

	response, _, err := ctx.API.CanvasChangeRequestAPI.
		CanvasesResolveCanvasChangeRequest(ctx.Context, canvasID, changeRequestID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	if response.ChangeRequest == nil {
		return nil
	}

	changeRequest := *response.ChangeRequest
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(changeRequest)
	}

	return renderCanvasChangeRequestSummaryText(ctx, "resolved", changeRequest)
}
