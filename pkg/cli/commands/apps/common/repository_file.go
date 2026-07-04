package common

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
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
