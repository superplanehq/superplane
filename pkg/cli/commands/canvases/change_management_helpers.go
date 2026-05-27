package canvases

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type changeManagementContext struct {
	changeManagementEnabled bool
}

func resolveChangeManagementContext(ctx core.CommandContext, canvasID string) (*changeManagementContext, error) {
	canvasResponse, _, err := ctx.API.CanvasAPI.
		CanvasesDescribeCanvas(ctx.Context, canvasID).
		Execute()
	if err != nil {
		return nil, err
	}
	if canvasResponse.Canvas == nil {
		return nil, fmt.Errorf("canvas not found")
	}

	spec := canvasResponse.Canvas.GetSpec()
	cm := spec.GetChangeManagement()
	return &changeManagementContext{
		changeManagementEnabled: cm.GetEnabled(),
	}, nil
}

func canvasFromVersion(version openapi_client.CanvasesCanvasVersion) openapi_client.CanvasesCanvas {
	canvas := openapi_client.CanvasesCanvas{}
	if version.Spec != nil {
		canvas.SetSpec(*version.Spec)
	}
	return canvas
}
