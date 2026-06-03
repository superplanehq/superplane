package common

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// FindAppID returns the app id for the given name or UUID. If
// `nameOrID` parses as a UUID, it is returned unchanged. Otherwise the
// CLI looks up apps by name and requires exactly one match.
func FindAppID(ctx core.CommandContext, client *openapi_client.APIClient, nameOrID string) (string, error) {
	if _, err := uuid.Parse(nameOrID); err == nil {
		return nameOrID, nil
	}

	return findAppIDByName(ctx, client, nameOrID)
}

// ResolveAppNameOrIDArg returns the app id for `arg`, falling back to
// the active app configured for the user when `arg` is empty. It returns
// a friendly error when neither is available so callers can surface the
// same message across commands.
func ResolveAppNameOrIDArg(ctx core.CommandContext, arg string) (string, error) {
	trimmed := strings.TrimSpace(arg)
	if trimmed == "" && ctx.Config != nil {
		trimmed = strings.TrimSpace(ctx.Config.GetActiveApp())
	}
	if trimmed == "" {
		return "", fmt.Errorf("<app-name-or-id> is required (or set an active app with `superplane apps active`)")
	}

	return FindAppID(ctx, ctx.API, trimmed)
}

// ChangeManagementEnabled reports whether change management is enabled on
// the app identified by `appID`.
func ChangeManagementEnabled(ctx core.CommandContext, appID string) (bool, error) {
	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, appID).Execute()
	if err != nil {
		return false, err
	}
	if response.Canvas == nil {
		return false, fmt.Errorf("app %q not found", appID)
	}

	spec := response.Canvas.GetSpec()
	cm := spec.GetChangeManagement()
	return cm.GetEnabled(), nil
}

func findAppIDByName(ctx core.CommandContext, client *openapi_client.APIClient, name string) (string, error) {
	response, _, err := client.CanvasAPI.CanvasesListCanvases(ctx.Context).Execute()
	if err != nil {
		return "", err
	}

	var matches []openapi_client.CanvasesCanvas
	for _, canvas := range response.GetCanvases() {
		if canvas.Metadata == nil || canvas.Metadata.Name == nil {
			continue
		}
		if *canvas.Metadata.Name == name {
			matches = append(matches, canvas)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("app %q not found", name)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("multiple apps named %q found", name)
	}

	if matches[0].Metadata == nil || matches[0].Metadata.Id == nil {
		return "", fmt.Errorf("app %q is missing an id", name)
	}

	return *matches[0].Metadata.Id, nil
}

// FindCurrentUserDraftVersionID returns the materialized commit SHA at the tip
// of the current user's default draft branch, or an empty string when none exists.
func FindCurrentUserDraftVersionID(ctx core.CommandContext, appID string) (string, error) {
	return FindCurrentUserDraftTipSHA(ctx, appID)
}

// EnsureCurrentUserDraftVersionID returns the tip commit SHA for the user's
// default draft branch, creating the branch when it does not yet exist.
func EnsureCurrentUserDraftVersionID(ctx core.CommandContext, appID string) (string, error) {
	return EnsureCurrentUserDraftTipSHA(ctx, appID)
}

// FindOwnedDraftVersionID returns the tip commit SHA for the user's default draft branch.
func FindOwnedDraftVersionID(ctx core.CommandContext, appID string, userID string) (string, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return "", nil
	}

	me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return "", err
	}
	currentUserID := strings.TrimSpace(me.User.GetId())
	if currentUserID == "" || !strings.EqualFold(currentUserID, trimmedUserID) {
		return "", nil
	}

	return FindCurrentUserDraftTipSHA(ctx, appID)
}

// DescribeAppVersionByID loads a specific app version and errors when
// the response does not include one.
func DescribeAppVersionByID(
	ctx core.CommandContext,
	appID string,
	versionID string,
) (openapi_client.CanvasesCanvasVersion, error) {
	response, _, err := ctx.API.CanvasVersionAPI.CanvasesDescribeCanvasVersion(ctx.Context, appID, versionID).Execute()
	if err != nil {
		return openapi_client.CanvasesCanvasVersion{}, err
	}
	if response.Version == nil {
		return openapi_client.CanvasesCanvasVersion{}, fmt.Errorf("app version %q not found", versionID)
	}

	return *response.Version, nil
}

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
