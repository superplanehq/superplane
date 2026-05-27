// Package canvasresolve groups the small set of helpers used by multiple
// CLI command packages to resolve a canvas (by name or id), read canvas
// settings such as change management, and find or create the current user's
// draft version. Keeping these in one place lets the `canvases` and
// `console` command packages share behavior without either depending on the
// other.
package canvasresolve

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// FindCanvasID returns the canvas id for the given name or UUID. If
// `nameOrID` parses as a UUID, it is returned unchanged. Otherwise the
// CLI looks up canvases by name and requires exactly one match.
func FindCanvasID(ctx core.CommandContext, client *openapi_client.APIClient, nameOrID string) (string, error) {
	if _, err := uuid.Parse(nameOrID); err == nil {
		return nameOrID, nil
	}

	return findCanvasIDByName(ctx, client, nameOrID)
}

// ResolveCanvasNameOrIDArg returns the canvas id for `arg`, falling back to
// the active canvas configured for the user when `arg` is empty. It returns
// a friendly error when neither is available so callers can surface the
// same message across commands.
func ResolveCanvasNameOrIDArg(ctx core.CommandContext, arg string) (string, error) {
	trimmed := strings.TrimSpace(arg)
	if trimmed == "" && ctx.Config != nil {
		trimmed = strings.TrimSpace(ctx.Config.GetActiveCanvas())
	}
	if trimmed == "" {
		return "", fmt.Errorf("<canvas-name-or-id> is required (or set an active canvas with `superplane canvases active`)")
	}

	return FindCanvasID(ctx, ctx.API, trimmed)
}

// ChangeManagementEnabled reports whether change management is enabled on
// the canvas identified by `canvasID`.
func ChangeManagementEnabled(ctx core.CommandContext, canvasID string) (bool, error) {
	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return false, err
	}
	if response.Canvas == nil {
		return false, fmt.Errorf("canvas %q not found", canvasID)
	}

	spec := response.Canvas.GetSpec()
	cm := spec.GetChangeManagement()
	return cm.GetEnabled(), nil
}

func findCanvasIDByName(ctx core.CommandContext, client *openapi_client.APIClient, name string) (string, error) {
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
		return "", fmt.Errorf("canvas %q not found", name)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("multiple canvases named %q found", name)
	}

	if matches[0].Metadata == nil || matches[0].Metadata.Id == nil {
		return "", fmt.Errorf("canvas %q is missing an id", name)
	}

	return *matches[0].Metadata.Id, nil
}

// FindCurrentUserDraftVersionID returns the id of the first non-published
// canvas version visible to the current user, or an empty string if none
// exists. It does not create a draft.
func FindCurrentUserDraftVersionID(ctx core.CommandContext, canvasID string) (string, error) {
	response, _, err := ctx.API.CanvasVersionAPI.CanvasesListCanvasVersions(ctx.Context, canvasID).Execute()
	if err != nil {
		return "", err
	}

	for _, version := range response.GetVersions() {
		metadata := version.GetMetadata()
		if metadata.GetState() == openapi_client.CANVASESCANVASVERSIONSTATE_STATE_PUBLISHED {
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

// EnsureCurrentUserDraftVersionID returns the id of the current user's draft
// version, creating one if it does not yet exist.
func EnsureCurrentUserDraftVersionID(ctx core.CommandContext, canvasID string) (string, error) {
	versionID, err := FindCurrentUserDraftVersionID(ctx, canvasID)
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

// FindOwnedDraftVersionID walks the version history (paginated) and returns
// the id of the latest non-published version whose owner matches `userID`,
// or an empty string when none is found.
func FindOwnedDraftVersionID(ctx core.CommandContext, canvasID string, userID string) (string, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return "", nil
	}

	var before *time.Time
	for {
		req := ctx.API.CanvasVersionAPI.
			CanvasesListCanvasVersions(ctx.Context, canvasID).
			Limit(50)
		if before != nil {
			req = req.Before(*before)
		}

		response, _, err := req.Execute()
		if err != nil {
			return "", err
		}

		for _, version := range response.GetVersions() {
			metadata := version.GetMetadata()
			if metadata.GetState() == openapi_client.CANVASESCANVASVERSIONSTATE_STATE_PUBLISHED {
				continue
			}

			ownerID := ""
			if metadata.Owner != nil {
				ownerID = strings.TrimSpace(metadata.Owner.GetId())
			}
			if ownerID == "" || !strings.EqualFold(ownerID, trimmedUserID) {
				continue
			}

			versionID := strings.TrimSpace(metadata.GetId())
			if versionID == "" {
				continue
			}

			return versionID, nil
		}

		if !response.GetHasNextPage() {
			return "", nil
		}

		last, ok := response.GetLastTimestampOk()
		if !ok || last == nil {
			return "", nil
		}
		before = last
	}
}

// DescribeCanvasVersionByID loads a specific canvas version and errors when
// the response does not include one.
func DescribeCanvasVersionByID(
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
