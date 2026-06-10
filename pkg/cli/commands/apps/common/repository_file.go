package common

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	CanvasYAMLRepositoryPath  = "canvas.yaml"
	ConsoleYAMLRepositoryPath = "console.yaml"
)

func FetchRepositoryFile(ctx core.CommandContext, canvasID, path, versionID string) ([]byte, error) {
	config := ctx.API.GetConfig()
	if config == nil {
		return nil, fmt.Errorf("api client config is required")
	}

	baseURL, err := config.ServerURLWithContext(ctx.Context, "CanvasRepositoryAPIService.CanvasesListCanvasRepositoryFiles")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("api_url is required")
	}

	values := url.Values{}
	values.Set("path", strings.TrimLeft(strings.TrimSpace(strings.ReplaceAll(path, "\\", "/")), "/"))
	if trimmedVersionID := strings.TrimSpace(versionID); trimmedVersionID != "" {
		values.Set("version_id", trimmedVersionID)
	}

	endpoint := fmt.Sprintf(
		"%s/api/v1/canvases/%s/repository/file?%s",
		strings.TrimRight(baseURL, "/"),
		url.PathEscape(canvasID),
		values.Encode(),
	)

	request, err := http.NewRequestWithContext(ctx.Context, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	if authorization := strings.TrimSpace(config.DefaultHeader["Authorization"]); authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(body))
		if message != "" {
			return nil, fmt.Errorf("%s", message)
		}
		return nil, fmt.Errorf("failed to read repository file: %s", response.Status)
	}

	return body, nil
}

// StageRepositorySpecFile writes a single canvas.yaml/console.yaml edit to the
// draft version's staging layer without touching the committed version row.
func StageRepositorySpecFile(
	ctx core.CommandContext,
	canvasID string,
	versionID string,
	path string,
	content []byte,
) error {
	operation := openapi_client.NewCanvasesCanvasRepositoryFileOperation()
	operation.SetPath(path)
	operation.SetContent(base64.StdEncoding.EncodeToString(content))

	body := openapi_client.NewCanvasesStageCanvasRepositoryFileBody()
	body.SetOperations([]openapi_client.CanvasesCanvasRepositoryFileOperation{*operation})

	_, _, err := ctx.API.CanvasRepositoryAPI.
		CanvasesStageCanvasRepositoryFile(ctx.Context, canvasID, versionID).
		Body(*body).
		Execute()
	return err
}

// CommitCanvasStaging parses the staged canvas.yaml/console.yaml into the draft
// version row and clears staging.
func CommitCanvasStaging(ctx core.CommandContext, canvasID, versionID string) error {
	_, _, err := ctx.API.CanvasVersionAPI.
		CanvasesCommitCanvasStaging(ctx.Context, canvasID, versionID).
		Body(map[string]any{}).
		Execute()
	return err
}

// ApplyCanvasAutoLayout lays out the staged canvas.yaml and re-stages it. A nil
// or unspecified-algorithm layout is treated as "no layout" and skipped.
func ApplyCanvasAutoLayout(
	ctx core.CommandContext,
	canvasID string,
	versionID string,
	autoLayout *openapi_client.CanvasesCanvasAutoLayout,
) error {
	if autoLayout == nil {
		return nil
	}
	if autoLayout.GetAlgorithm() == openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_UNSPECIFIED {
		return nil
	}

	body := openapi_client.NewCanvasesApplyCanvasAutoLayoutBody()
	body.SetAutoLayout(*autoLayout)

	_, _, err := ctx.API.CanvasVersionAPI.
		CanvasesApplyCanvasAutoLayout(ctx.Context, canvasID, versionID).
		Body(*body).
		Execute()
	return err
}

// StageCommitRepositorySpecFile stages a spec file, optionally lays out the
// staged canvas, then commits staging into the draft version row.
func StageCommitRepositorySpecFile(
	ctx core.CommandContext,
	canvasID string,
	versionID string,
	path string,
	content []byte,
	autoLayout *openapi_client.CanvasesCanvasAutoLayout,
) error {
	if err := StageRepositorySpecFile(ctx, canvasID, versionID, path, content); err != nil {
		return err
	}

	if path == CanvasYAMLRepositoryPath {
		if err := ApplyCanvasAutoLayout(ctx, canvasID, versionID, autoLayout); err != nil {
			return err
		}
	}

	return CommitCanvasStaging(ctx, canvasID, versionID)
}
