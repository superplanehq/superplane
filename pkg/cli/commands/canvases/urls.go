package canvases

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

// BuildCanvasURL composes the canonical web URL for a canvas. Returns "" when
// the context, base URL, organization id, or canvas id is missing so callers
// can omit the URL output without erroring.
//
// orgID and canvasID should come from the API response (canvas metadata)
// rather than the local CLI context, so the URL stays correct even if the
// active context drifts.
func BuildCanvasURL(ctx core.CommandContext, orgID, canvasID string) string {
	if ctx.Config == nil || orgID == "" || canvasID == "" {
		return ""
	}

	baseURL := ctx.Config.GetURL()
	if baseURL == "" {
		return ""
	}

	return fmt.Sprintf("%s/%s/canvases/%s", baseURL, orgID, canvasID)
}
