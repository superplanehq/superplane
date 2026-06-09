package common

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const multipleDraftsError = "multiple drafts found; pass --draft-id (see `superplane apps drafts list`)"

// MergeDraftOrVersionID resolves --draft-id and --version-id when both refer to
// the same version id. Returns an error when both are set to different values.
func MergeDraftOrVersionID(draftID, versionID *string) (string, error) {
	resolvedDraftID := ""
	resolvedVersionID := ""
	if draftID != nil {
		resolvedDraftID = strings.TrimSpace(*draftID)
	}
	if versionID != nil {
		resolvedVersionID = strings.TrimSpace(*versionID)
	}

	if resolvedDraftID != "" && resolvedVersionID != "" && !strings.EqualFold(resolvedDraftID, resolvedVersionID) {
		return "", fmt.Errorf("--draft-id and --version-id must match when both are set")
	}
	if resolvedDraftID != "" {
		return resolvedDraftID, nil
	}
	return resolvedVersionID, nil
}

// ListDraftVersions returns all draft versions for an app.
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

// ListOwnedDraftVersions returns draft versions owned by userID.
func ListOwnedDraftVersions(ctx core.CommandContext, appID, userID string) ([]openapi_client.CanvasesCanvasVersion, error) {
	versions, err := ListDraftVersions(ctx, appID)
	if err != nil {
		return nil, err
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return nil, nil
	}

	var owned []openapi_client.CanvasesCanvasVersion
	for _, version := range versions {
		ownerID := ""
		if version.Metadata != nil && version.Metadata.Owner != nil {
			ownerID = strings.TrimSpace(version.Metadata.Owner.GetId())
		}
		if ownerID == "" || !strings.EqualFold(ownerID, trimmedUserID) {
			continue
		}
		owned = append(owned, version)
	}

	return owned, nil
}

// DraftResolveOptions controls how ResolveDraftVersionID picks a draft version.
type DraftResolveOptions struct {
	DraftID     string
	UseDraft    bool
	AllowCreate bool
}

// ResolveDraftVersionID selects a draft version according to the selector rules
// in issue #5237. When UseDraft is false and DraftID is empty, it returns ("", nil)
// so callers can fall back to live reads or EnsureCurrentUserDraftVersionID.
func ResolveDraftVersionID(ctx core.CommandContext, appID string, opts DraftResolveOptions) (string, error) {
	if !opts.UseDraft && strings.TrimSpace(opts.DraftID) == "" {
		return "", nil
	}

	currentUserID, err := currentUserID(ctx)
	if err != nil {
		return "", err
	}

	draftID := strings.TrimSpace(opts.DraftID)
	if draftID != "" {
		return validateOwnedDraftVersion(ctx, appID, draftID, currentUserID)
	}

	owned, err := ListOwnedDraftVersions(ctx, appID, currentUserID)
	if err != nil {
		return "", err
	}

	switch len(owned) {
	case 0:
		if opts.AllowCreate {
			return createDraftVersion(ctx, appID, "")
		}
		return "", fmt.Errorf("draft version not found for current user")
	case 1:
		return versionIDFromMetadata(owned[0]), nil
	default:
		return "", fmt.Errorf("%s", multipleDraftsError)
	}
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

func versionIDFromMetadata(version openapi_client.CanvasesCanvasVersion) string {
	if version.Metadata == nil {
		return ""
	}
	return strings.TrimSpace(version.Metadata.GetId())
}
