package core

import "fmt"

// TODO: make the base URL configurable via config/env
const appBaseURL = "https://app.superplane.com"

func CanvasURL(ctx CommandContext, canvasID string) string {
	orgID, err := ResolveOrganizationID(ctx)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s/%s/canvases/%s", appBaseURL, orgID, canvasID)
}
