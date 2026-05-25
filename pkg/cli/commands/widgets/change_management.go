package widgets

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// findCurrentUserDraftVersionID returns the current user's draft version
// id, or "" if no draft exists yet. Mirrors the canvases package helper.
func findCurrentUserDraftVersionID(ctx core.CommandContext, canvasID string) (string, error) {
	response, _, err := ctx.API.CanvasVersionAPI.CanvasesListCanvasVersions(ctx.Context, canvasID).Execute()
	if err != nil {
		return "", err
	}

	for _, version := range response.GetVersions() {
		metadata := version.GetMetadata()
		if metadata.GetState() == openapi_client.CANVASESCANVASVERSIONSTATE_STATE_PUBLISHED {
			continue
		}
		if id := strings.TrimSpace(metadata.GetId()); id != "" {
			return id, nil
		}
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
		Body(map[string]any{}).
		Execute()
	if err != nil {
		return "", err
	}
	if response.Version == nil || response.Version.Metadata == nil {
		return "", fmt.Errorf("draft version was not returned by the API")
	}

	id := strings.TrimSpace(response.Version.Metadata.GetId())
	if id == "" {
		return "", fmt.Errorf("draft version id was not returned by the API")
	}
	return id, nil
}

// changeManagementEnabled tells callers whether the canvas requires going
// through the change-request flow rather than auto-publishing.
func changeManagementEnabled(ctx core.CommandContext, canvasID string) (bool, error) {
	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return false, err
	}
	if response.Canvas == nil {
		return false, fmt.Errorf("canvas not found")
	}
	spec := response.Canvas.GetSpec()
	cm := spec.GetChangeManagement()
	return cm.GetEnabled(), nil
}

// canvasFromVersion returns a canvas containing the given version's spec
// (the wire shape required by the update-version endpoint).
func canvasFromVersion(version openapi_client.CanvasesCanvasVersion) openapi_client.CanvasesCanvas {
	canvas := openapi_client.CanvasesCanvas{}
	if version.Spec != nil {
		canvas.SetSpec(*version.Spec)
	}
	return canvas
}

// describeCanvasVersionByID reads a specific canvas version, used so the
// CLI can mutate the same draft the API would publish on auto-publish.
func describeCanvasVersionByID(
	ctx core.CommandContext,
	canvasID, versionID string,
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

// updateAndMaybePublish persists the new canvas spec into the user's draft
// version and (when not in --draft mode) publishes the draft so the changes
// land in the live canvas. Errors from the publish step include the prefix
// "draft was updated but publish failed" so users know they can re-run a
// `superplane canvases change-requests publish` instead of starting over.
func updateAndMaybePublish(
	ctx core.CommandContext,
	canvasID, versionID string,
	canvas openapi_client.CanvasesCanvas,
	draftMode bool,
) (openapi_client.CanvasesCanvasVersion, error) {
	body := openapi_client.CanvasesUpdateCanvasVersionBody{}
	body.SetCanvas(canvas)
	body.SetVersionId(versionID)

	response, _, err := ctx.API.CanvasVersionAPI.
		CanvasesUpdateCanvasVersion2(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return openapi_client.CanvasesCanvasVersion{}, err
	}
	version := response.GetVersion()

	if draftMode {
		return version, nil
	}

	if _, _, err := ctx.API.CanvasVersionAPI.
		CanvasesPublishCanvasVersion(ctx.Context, canvasID, versionID).
		Body(map[string]any{}).
		Execute(); err != nil {
		return version, fmt.Errorf("draft was updated but publish failed: %w", err)
	}
	return version, nil
}
