package canvases

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type canvasVersioningContext struct {
	versioningEnabled bool
}

func resolveCanvasVersioningContext(ctx core.CommandContext, canvasID string) (*canvasVersioningContext, error) {
	canvasResponse, _, err := ctx.API.CanvasAPI.
		CanvasesDescribeCanvas(ctx.Context, canvasID).
		Execute()
	if err != nil {
		return nil, err
	}
	if canvasResponse.Canvas == nil || canvasResponse.Canvas.Metadata == nil {
		return nil, fmt.Errorf("canvas metadata not found")
	}

	return &canvasVersioningContext{
		versioningEnabled: canvasResponse.Canvas.Metadata.GetVersioningEnabled(),
	}, nil
}

func findCurrentUserDraftVersionID(ctx core.CommandContext, canvasID string) (string, error) {
	response, _, err := ctx.API.CanvasVersionAPI.CanvasesListCanvasVersions(ctx.Context, canvasID).Execute()
	if err != nil {
		return "", err
	}

	for _, version := range response.GetVersions() {
		metadata := version.GetMetadata()
		if metadata.GetIsPublished() {
			continue
		}

		versionID := strings.TrimSpace(metadata.GetId())
		if versionID == "" {
			continue
		}

		return versionID, nil
	}

	return "", nil
}

func ensureCurrentUserDraftVersionID(ctx core.CommandContext, canvasID string) (string, error) {
	versionID, err := findCurrentUserDraftVersionID(ctx, canvasID)
	if err != nil {
		return "", err
	}
	if versionID != "" {
		return versionID, nil
	}

	response, _, err := ctx.API.CanvasVersionAPI.
		CanvasesCreateCanvasVersion(ctx.Context, canvasID).
		Body(map[string]interface{}{}).
		Execute()
	if err != nil {
		return "", err
	}
	if response.Version == nil || response.Version.Metadata == nil {
		return "", fmt.Errorf("draft version was not returned by the API")
	}

	versionID = strings.TrimSpace(response.Version.Metadata.GetId())
	if versionID == "" {
		return "", fmt.Errorf("draft version id was not returned by the API")
	}

	return versionID, nil
}

func describeCanvasVersionByID(
	ctx core.CommandContext,
	canvasID string,
	versionID string,
) (openapi_client.CanvasesCanvasVersion, error) {
	response, _, err := ctx.API.CanvasVersionAPI.CanvasesDescribeCanvasVersion(ctx.Context, canvasID, versionID).Execute()
	if err != nil {
		return openapi_client.CanvasesCanvasVersion{}, err
	}
	if response.Version == nil {
		return openapi_client.CanvasesCanvasVersion{}, fmt.Errorf("canvas version %q not found", versionID)
	}

	return *response.Version, nil
}

func canvasFromVersion(version openapi_client.CanvasesCanvasVersion) openapi_client.CanvasesCanvas {
	canvas := openapi_client.CanvasesCanvas{}
	if version.Spec != nil {
		canvas.SetSpec(*version.Spec)
	}
	return canvas
}
