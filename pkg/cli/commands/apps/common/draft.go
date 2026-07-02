package common

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// EnsureLiveVersionID returns the newest committed version id for an app.
func EnsureLiveVersionID(ctx core.CommandContext, appID string) (string, error) {
	return ResolveLiveVersionID(ctx, appID, "")
}

// ResolveLiveVersionID validates an optional explicit version id against the app
// history and returns the resolved live version id.
func ResolveLiveVersionID(ctx core.CommandContext, appID, versionID string) (string, error) {
	trimmedVersionID := strings.TrimSpace(versionID)
	if trimmedVersionID != "" {
		version, err := DescribeAppVersionByID(ctx, appID, trimmedVersionID)
		if err != nil {
			return "", err
		}
		if version.Metadata == nil || strings.TrimSpace(version.Metadata.GetId()) == "" {
			return "", fmt.Errorf("version %q not found", trimmedVersionID)
		}
		return strings.TrimSpace(version.Metadata.GetId()), nil
	}

	response, _, err := ctx.API.CanvasVersionAPI.
		CanvasesListCanvasVersions(ctx.Context, appID).
		Limit(1).
		Execute()
	if err != nil {
		return "", err
	}

	versions := response.GetVersions()
	if len(versions) == 0 || versions[0].Metadata == nil {
		return "", fmt.Errorf("app %q has no committed versions", appID)
	}

	liveVersionID := strings.TrimSpace(versions[0].Metadata.GetId())
	if liveVersionID == "" {
		return "", fmt.Errorf("live version id was not returned by the API")
	}

	return liveVersionID, nil
}

// GetCanvasStaging loads the current user's staging summary for an app.
func GetCanvasStaging(ctx core.CommandContext, appID string) (openapi_client.CanvasesStagingSummary, error) {
	response, _, err := ctx.API.CanvasStagingAPI.
		CanvasesGetCanvasStaging(ctx.Context, appID).
		Execute()
	if err != nil {
		return openapi_client.CanvasesStagingSummary{}, err
	}
	return response.GetStagingSummary(), nil
}

// StageRepositorySpecFile writes a repository spec file into per-user staging.
func StageRepositorySpecFile(
	ctx core.CommandContext,
	canvasID string,
	path string,
	content []byte,
) error {
	operation := openapi_client.NewCanvasesCanvasRepositoryFileOperation()
	operation.SetPath(path)
	operation.SetContent(base64.StdEncoding.EncodeToString(content))

	body := openapi_client.NewCanvasesPutCanvasStagingBody()
	body.SetOperations([]openapi_client.CanvasesCanvasRepositoryFileOperation{*operation})

	_, _, err := ctx.API.CanvasStagingAPI.
		CanvasesPutCanvasStaging(ctx.Context, canvasID).
		Body(*body).
		Execute()
	return err
}

// CommitCanvasStaging commits staged edits to main with the given message.
func CommitCanvasStaging(ctx core.CommandContext, canvasID, commitMessage string) (*openapi_client.CanvasesCommitCanvasStagingResponse, error) {
	body := openapi_client.NewCanvasesCommitCanvasStagingBody()
	body.SetCommitMessage(strings.TrimSpace(commitMessage))

	response, _, err := ctx.API.CanvasStagingAPI.
		CanvasesCommitCanvasStaging(ctx.Context, canvasID).
		Body(*body).
		Execute()
	return response, err
}

// DiscardCanvasStaging discards all staged edits for the current user.
func DiscardCanvasStaging(ctx core.CommandContext, canvasID string) error {
	_, _, err := ctx.API.CanvasStagingAPI.
		CanvasesDeleteCanvasStaging(ctx.Context, canvasID).
		Execute()
	return err
}

// FindCurrentUserDraftVersionID returns the live version id for an app.
func FindCurrentUserDraftVersionID(ctx core.CommandContext, appID string) (string, error) {
	return EnsureLiveVersionID(ctx, appID)
}

// EnsureCurrentUserDraftVersionID is kept for CLI compatibility and resolves the live version id.
func EnsureCurrentUserDraftVersionID(ctx core.CommandContext, appID string) (string, error) {
	return EnsureLiveVersionID(ctx, appID)
}

// ResolveDraftVersionID is kept for CLI compatibility and validates an explicit version id.
func ResolveDraftVersionID(ctx core.CommandContext, appID, draftID string) (string, error) {
	trimmedDraftID := strings.TrimSpace(draftID)
	if trimmedDraftID == "" {
		return "", fmt.Errorf("version id is required")
	}
	return ResolveLiveVersionID(ctx, appID, trimmedDraftID)
}
