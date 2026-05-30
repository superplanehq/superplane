package canvas

import (
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func canvasFromVersion(version openapi_client.CanvasesCanvasVersion) openapi_client.CanvasesCanvas {
	canvas := openapi_client.CanvasesCanvas{}
	if version.Spec != nil {
		canvas.SetSpec(*version.Spec)
	}
	return canvas
}
