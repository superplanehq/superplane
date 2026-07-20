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

func findAppIDByName(ctx core.CommandContext, client *openapi_client.APIClient, name string) (string, error) {
	response, _, err := client.CanvasAPI.CanvasesListCanvases(ctx.Context).Execute()
	if err != nil {
		return "", err
	}

	var matches []openapi_client.CanvasesCanvasSummary
	for _, canvas := range response.GetCanvases() {
		if canvas.Name == nil {
			continue
		}
		if *canvas.Name == name {
			matches = append(matches, canvas)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("app %q not found", name)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("multiple apps named %q found", name)
	}

	if matches[0].Id == nil {
		return "", fmt.Errorf("app %q is missing an id", name)
	}

	return *matches[0].Id, nil
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

// BuildCanvasURL composes the canonical web URL for an app. Returns "" when
// the context, base URL, organization id, or app id is missing so callers
// can omit the URL output without erroring.
//
// orgID and canvasID should come from the API response (canvas metadata)
// rather than the local CLI context, so the URL stays correct even if the
// active context drifts.
func BuildAppURL(ctx core.CommandContext, orgID, canvasID string) string {
	if ctx.Config == nil || orgID == "" || canvasID == "" {
		return ""
	}

	baseURL := ctx.Config.GetURL()
	if baseURL == "" {
		return ""
	}

	return fmt.Sprintf("%s/%s/apps/%s", baseURL, orgID, canvasID)
}
