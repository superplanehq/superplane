package changes

import "github.com/superplanehq/superplane/pkg/cli/core"

type GetCommand struct{}

func (c *GetCommand) Execute(ctx core.CommandContext) error {
	changeRequestID, canvasTarget, err := parseCanvasChangeRequestTargetArgs(ctx.Args)
	if err != nil {
		return err
	}

	canvasID, err := resolveCanvasTargetFromOptionalArg(ctx, canvasTarget)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasChangeRequestAPI.
		CanvasesDescribeCanvasChangeRequest(ctx.Context, canvasID, changeRequestID).
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

	return renderCanvasChangeRequestText(ctx, changeRequest)
}
