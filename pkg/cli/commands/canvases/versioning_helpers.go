package canvases

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type canvasVersioningContext struct {
	currentUserID      string
	sandboxModeEnabled bool
}

func resolveCanvasVersioningContext(ctx core.CommandContext) (*canvasVersioningContext, error) {
	me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return nil, err
	}

	currentUserID := strings.TrimSpace(me.GetId())
	if currentUserID == "" {
		return nil, fmt.Errorf("user id not found for authenticated user")
	}

	organizationID := strings.TrimSpace(me.GetOrganizationId())
	if organizationID == "" {
		return nil, fmt.Errorf("organization id not found for authenticated user")
	}

	organizationResponse, _, err := ctx.API.OrganizationAPI.
		OrganizationsDescribeOrganization(ctx.Context, organizationID).
		Execute()
	if err != nil {
		return nil, err
	}
	if organizationResponse.Organization == nil || organizationResponse.Organization.Metadata == nil {
		return nil, fmt.Errorf("organization metadata not found")
	}

	return &canvasVersioningContext{
		currentUserID:      currentUserID,
		sandboxModeEnabled: organizationResponse.Organization.Metadata.GetCanvasSandboxModeEnabled(),
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
