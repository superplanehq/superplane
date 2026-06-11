package common

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// ListDraftVersions returns the current user's draft versions for an app.
// The API scopes drafts to the authenticated user, so no client-side owner
// filtering is required.
func ListDraftVersions(ctx core.CommandContext, appID string) ([]openapi_client.CanvasesCanvasVersion, error) {
	response, _, err := ctx.API.CanvasVersionAPI.
		CanvasesListCanvasVersions(ctx.Context, appID).
		State("STATE_DRAFT").
		Execute()
	if err != nil {
		return nil, err
	}

	return response.GetVersions(), nil
}

// ResolveDraftVersionID validates that draftID refers to a draft version owned
// by the current user and returns it. It errors when the id is empty, points to
// a non-draft version, or is owned by another user.
func ResolveDraftVersionID(ctx core.CommandContext, appID, draftID string) (string, error) {
	trimmedDraftID := strings.TrimSpace(draftID)
	if trimmedDraftID == "" {
		return "", fmt.Errorf("draft id is required")
	}

	currentUserID, err := currentUserID(ctx)
	if err != nil {
		return "", err
	}

	return validateOwnedDraftVersion(ctx, appID, trimmedDraftID, currentUserID)
}

func currentUserID(ctx core.CommandContext) (string, error) {
	me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return "", err
	}

	currentUserID := strings.TrimSpace(me.User.GetId())
	if currentUserID == "" {
		return "", fmt.Errorf("current user id not found")
	}

	return currentUserID, nil
}

func validateOwnedDraftVersion(ctx core.CommandContext, appID, draftID, currentUserID string) (string, error) {
	version, err := DescribeAppVersionByID(ctx, appID, draftID)
	if err != nil {
		return "", err
	}

	if version.Metadata == nil {
		return "", fmt.Errorf("draft %q not found", draftID)
	}

	state := strings.TrimSpace(string(version.Metadata.GetState()))
	if state != "" && state != "STATE_DRAFT" {
		return "", fmt.Errorf("version %q is not a draft", draftID)
	}

	ownerID := ""
	if version.Metadata.Owner != nil {
		ownerID = strings.TrimSpace(version.Metadata.Owner.GetId())
	}
	if ownerID == "" || !strings.EqualFold(ownerID, currentUserID) {
		return "", fmt.Errorf("draft %q is not owned by the current user", draftID)
	}

	return draftID, nil
}

func createDraftVersion(ctx core.CommandContext, appID, displayName string) (string, error) {
	body := openapi_client.CanvasesCreateCanvasVersionBody{}
	trimmedName := strings.TrimSpace(displayName)
	if trimmedName != "" {
		body.SetDisplayName(trimmedName)
	}

	response, _, err := ctx.API.CanvasVersionAPI.
		CanvasesCreateCanvasVersion(ctx.Context, appID).
		Body(body).
		Execute()
	if err != nil {
		return "", err
	}
	if response.Version == nil || response.Version.Metadata == nil {
		return "", fmt.Errorf("draft version was not returned by the API")
	}

	versionID := strings.TrimSpace(response.Version.Metadata.GetId())
	if versionID == "" {
		return "", fmt.Errorf("draft version id was not returned by the API")
	}

	return versionID, nil
}
