package common

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
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

	response, _, err := ctx.API.CanvasAPI.
		CanvasesDescribeCanvas(ctx.Context, appID).
		Execute()
	if err != nil {
		return "", err
	}
	if response.Canvas == nil || response.Canvas.Metadata == nil {
		return "", fmt.Errorf("app %q was not found", appID)
	}

	liveVersionID := strings.TrimSpace(response.Canvas.Metadata.GetVersionId())
	if liveVersionID == "" {
		return "", fmt.Errorf("app %q has no committed versions", appID)
	}

	return liveVersionID, nil
}

// GetCanvasStaging loads the current user's staging state for an app.
func GetCanvasStaging(ctx core.CommandContext, appID string) (openapi_client.CanvasesStaging, error) {
	response, _, err := ctx.API.CanvasStagingAPI.
		CanvasesGetCanvasStaging(ctx.Context, appID).
		Execute()
	if err != nil {
		return openapi_client.CanvasesStaging{}, err
	}
	return response.GetStaging(), nil
}

// NormalizeRepositoryPath returns a repository-relative path.
func NormalizeRepositoryPath(path string) string {
	return strings.TrimLeft(strings.TrimSpace(strings.ReplaceAll(path, "\\", "/")), "/")
}

// RepositoryPathFromLocalFile maps a local file path to its repository path.
func RepositoryPathFromLocalFile(localPath string) string {
	return NormalizeRepositoryPath(filepath.Base(localPath))
}

// RequireCommitMessage validates a commit message flag value.
func RequireCommitMessage(message string) (string, error) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "", fmt.Errorf("--message is required")
	}
	return trimmed, nil
}

// StageRepositorySpecFile writes a repository spec file into per-user staging.
func StageRepositorySpecFile(
	ctx core.CommandContext,
	canvasID string,
	path string,
	content []byte,
) error {
	return StageRepositoryFiles(ctx, canvasID, []RepositoryFileStaging{{
		Path:    path,
		Content: content,
	}})
}

type RepositoryFileStaging struct {
	Path    string
	Content []byte
}

// StageRepositoryFiles writes one or more repository files into per-user staging.
func StageRepositoryFiles(ctx core.CommandContext, canvasID string, files []RepositoryFileStaging) error {
	if len(files) == 0 {
		return fmt.Errorf("at least one file is required")
	}

	operations := make([]openapi_client.CanvasesCanvasRepositoryFileOperation, 0, len(files))
	for _, file := range files {
		path := NormalizeRepositoryPath(file.Path)
		if path == "" {
			return fmt.Errorf("repository path is required")
		}

		operation := openapi_client.NewCanvasesCanvasRepositoryFileOperation()
		operation.SetPath(path)
		operation.SetContent(base64.StdEncoding.EncodeToString(file.Content))
		operations = append(operations, *operation)
	}

	body := openapi_client.NewCanvasesPutCanvasStagingBody()
	body.SetOperations(operations)

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
