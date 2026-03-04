package canvases

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func resolveCanvasIDFromArgOrActive(ctx core.CommandContext, canvasRef string) (string, error) {
	trimmed := strings.TrimSpace(canvasRef)
	if trimmed != "" {
		return findCanvasID(ctx, ctx.API, trimmed)
	}

	if ctx.Config == nil {
		return "", fmt.Errorf("canvas id is required; pass one or set an active canvas with \"superplane canvases active\"")
	}

	activeCanvas := strings.TrimSpace(ctx.Config.GetActiveCanvas())
	if activeCanvas == "" {
		return "", fmt.Errorf("canvas id is required; pass one or set an active canvas with \"superplane canvases active\"")
	}

	return findCanvasID(ctx, ctx.API, activeCanvas)
}

func resolveWorkingVersionIDFromArgOrActive(ctx core.CommandContext, versionRef string) (string, error) {
	trimmed := strings.TrimSpace(versionRef)
	if trimmed == "" {
		if ctx.Config == nil {
			return "", fmt.Errorf("edit version id is required; pass one or select one with \"superplane canvases versions use\"")
		}

		activeVersion := strings.TrimSpace(ctx.Config.GetActiveCanvasVersion())
		if activeVersion == "" {
			return "", fmt.Errorf("edit version id is required; pass one or select one with \"superplane canvases versions use\"")
		}
		return activeVersion, nil
	}

	if strings.EqualFold(trimmed, "live") {
		return "", fmt.Errorf("live is read-only; create an edit version first")
	}

	return trimmed, nil
}

func resolveVersionRef(ctx core.CommandContext, canvasID string, versionRef string) (string, bool, error) {
	trimmed := strings.TrimSpace(versionRef)
	if trimmed == "" {
		if ctx.Config == nil {
			return "", false, fmt.Errorf("version id is required; pass one or select one with \"superplane canvases versions use\"")
		}

		activeVersion := strings.TrimSpace(ctx.Config.GetActiveCanvasVersion())
		if activeVersion == "" {
			return "", false, fmt.Errorf("version id is required; pass one or select one with \"superplane canvases versions use\"")
		}
		return activeVersion, false, nil
	}

	if strings.EqualFold(trimmed, "live") {
		versionID, err := findLiveCanvasVersionID(ctx, canvasID)
		if err != nil {
			return "", false, err
		}
		return versionID, true, nil
	}

	return trimmed, false, nil
}

func findLiveCanvasVersionID(ctx core.CommandContext, canvasID string) (string, error) {
	response, _, err := ctx.API.CanvasVersionAPI.
		CanvasesListCanvasVersions(ctx.Context, canvasID).
		Execute()
	if err != nil {
		return "", err
	}

	for _, version := range response.GetVersions() {
		metadata := version.GetMetadata()
		if metadata.GetIsPublished() {
			return metadata.GetId(), nil
		}
	}

	return "", fmt.Errorf("live version not found for canvas %q", canvasID)
}

func describeCanvasVersion(ctx core.CommandContext, canvasID string, versionID string) (openapi_client.CanvasesCanvasVersion, error) {
	response, _, err := ctx.API.CanvasVersionAPI.
		CanvasesDescribeCanvasVersion(ctx.Context, canvasID, versionID).
		Execute()
	if err != nil {
		return openapi_client.CanvasesCanvasVersion{}, err
	}

	if response.Version == nil {
		return openapi_client.CanvasesCanvasVersion{}, fmt.Errorf("version %q not found", versionID)
	}

	return *response.Version, nil
}

func isSandboxModeDisabledError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(strings.ToLower(err.Error()), "sandbox mode is disabled")
}

func setActiveCanvasAndVersion(ctx core.CommandContext, canvasID string, versionID string) error {
	if ctx.Config == nil {
		return nil
	}

	if err := ctx.Config.SetActiveCanvas(canvasID); err != nil {
		return err
	}

	return ctx.Config.SetActiveCanvasVersion(versionID)
}
