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

func CommitRepositoryFiles(
	ctx core.CommandContext,
	canvasID string,
	versionID string,
	expectedHeadSHA string,
	message string,
	operations []openapi_client.CanvasesCanvasRepositoryFileOperation,
	autoLayout *openapi_client.CanvasesCanvasAutoLayout,
	includeAutoLayout bool,
) (*openapi_client.CanvasesCommitCanvasRepositoryFilesResponse, error) {
	body := openapi_client.NewCanvasesCommitCanvasRepositoryFilesBody()
	if trimmedVersionID := strings.TrimSpace(versionID); trimmedVersionID != "" {
		body.SetVersionId(trimmedVersionID)
	}
	if trimmedHead := strings.TrimSpace(expectedHeadSHA); trimmedHead != "" {
		body.SetExpectedHeadSha(trimmedHead)
	}
	if trimmedMessage := strings.TrimSpace(message); trimmedMessage != "" {
		body.SetMessage(trimmedMessage)
	}
	body.SetOperations(operations)
	if includeAutoLayout {
		if autoLayout != nil {
			body.SetAutoLayout(*autoLayout)
		} else {
			body.SetAutoLayout(openapi_client.CanvasesCanvasAutoLayout{})
		}
	}

	response, _, err := ctx.API.CanvasRepositoryAPI.
		CanvasesCommitCanvasRepositoryFiles(ctx.Context, canvasID).
		Body(*body).
		Execute()
	return response, err
}

func CommitRepositorySpecFile(
	ctx core.CommandContext,
	canvasID string,
	versionID string,
	path string,
	content []byte,
	message string,
	autoLayout *openapi_client.CanvasesCanvasAutoLayout,
	includeAutoLayout bool,
) error {
	operation := openapi_client.NewCanvasesCanvasRepositoryFileOperation()
	operation.SetPath(path)
	operation.SetContent(base64.StdEncoding.EncodeToString(content))

	_, err := CommitRepositoryFiles(
		ctx,
		canvasID,
		versionID,
		"",
		message,
		[]openapi_client.CanvasesCanvasRepositoryFileOperation{*operation},
		autoLayout,
		includeAutoLayout,
	)
	return err
}
